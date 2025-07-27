package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/kode-ai/types"
)

// serverSession handles WebSocket server communication
type serverSession struct {
	stream        types.StreamContext
	eventCallback types.EventCallback
	logger        types.Logger

	lastAssistantMsg string
}

// ChatWithServer connects to a WebSocket chat server and streams events until finished
func ChatWithServer(ctx context.Context, server string, req types.Request) (*types.Response, error) {
	sess := &serverSession{}
	return sess.chatWithServer(ctx, server, req)
}

// chatWithServer connects to a WebSocket server and handles the streaming protocol
func (c *serverSession) chatWithServer(ctx context.Context, server string, req types.Request) (*types.Response, error) {
	c.eventCallback = req.EventCallback
	c.logger = getLogger(req.Logger)

	// Parse server URL and build WebSocket URL
	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Convert http/https to ws/wss
	scheme := "ws"
	if serverURL.Scheme == "https" {
		scheme = "wss"
	}

	// Build WebSocket URL with query parameters
	wsURL := &url.URL{
		Scheme: scheme,
		Host:   serverURL.Host,
		Path:   "/stream",
	}

	query := wsURL.Query()
	query.Set("wait_for_stream_events", "true")
	wsURL.RawQuery = query.Encode()

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, wsURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer conn.Close()

	c.stream = &websocketStreamContext{conn: conn}

	initReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal init request: %w", err)
	}

	// write entire request as init request
	if err := c.writeEvent(types.Message{
		Type:    types.MsgType_StreamInitRequest,
		Content: string(initReq),
	}); err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to write init request: %v\n", err)
		return nil, fmt.Errorf("failed to write init request: %w", err)
	}

	// Send finish marker for initial events
	if err := c.writeEvent(types.Message{
		Type: types.MsgType_StreamInitEventsFinished,
	}); err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to write finish marker: %v\n", err)
		return nil, fmt.Errorf("failed to write finish marker: %w", err)
	}

	// Process WebSocket messages
	return c.processWebSocketMessages(ctx, conn, req.ToolCallback, req.FollowUpCallback, req.ToolDefinitions)
}

// writeEvent writes an event and calls the event callback
func (c *serverSession) writeEvent(msg types.Message) error {
	// Call event callback first
	if c.eventCallback != nil {
		c.eventCallback(msg)
	}

	// Then write to stream
	if c.stream != nil {
		return c.stream.Write(msg)
	}
	return fmt.Errorf("no stream context available")
}

// processWebSocketMessages processes messages from the WebSocket connection
func (c *serverSession) processWebSocketMessages(ctx context.Context, conn *websocket.Conn, toolCallback types.ToolCallback, followUpCallback types.FollowUpCallback, toolDefs []*types.UnifiedTool) (*types.Response, error) {
	var response types.Response

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var msg types.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break // Normal close
			}
			return nil, fmt.Errorf("failed to read WebSocket message: %w", err)
		}

		if msg.Type == types.MsgType_Msg && msg.Role == types.Role_Assistant {
			c.lastAssistantMsg = msg.Content
		}

		if c.eventCallback != nil {
			c.eventCallback(msg)
		}

		// Handle stream requests if we have stream context
		if c.stream != nil {
			var unableToHandle bool
			switch msg.Type {
			case types.MsgType_StreamRequestTool:
				// Find tool callback
				var foundToolCallback types.ToolCallback
				if msg.ToolName != "" && msg.Content != "" {
					for _, toolDef := range toolDefs {
						if toolDef.Name == msg.ToolName {
							if toolDef.Handle != nil {
								foundToolCallback = toolDef.Handle
							} else if len(toolDef.Command) > 0 {
								foundToolCallback = makeCmdToolCallback(toolDef)
							}
							break
						}
					}
				}
				if foundToolCallback == nil {
					foundToolCallback = toolCallback
				}
				if foundToolCallback != nil {
					c.handleSingleToolCallback(ctx, msg, foundToolCallback)
					continue
				}
				unableToHandle = true
			case types.MsgType_StreamRequestUserMsg:
				if followUpCallback != nil {
					// ack
					err := c.stream.ACK(msg.StreamID)
					if err != nil {
						c.logger.Log(ctx, types.LogType_Error, "failed to ack stream: %v\n", err)
						continue
					}
					followUpResult, err := followUpCallback(ctx)
					if err != nil {
						c.writeEvent(types.Message{
							Type:     types.MsgType_Error,
							StreamID: msg.StreamID,
							Error:    err.Error(),
						})
						continue
					}
					if followUpResult != nil {
						fmsg := *followUpResult
						if fmsg.Type == "" {
							fmsg.Type = types.MsgType_Msg
						}
						if fmsg.Role == "" {
							fmsg.Role = types.Role_User
						}
						fmsg.StreamID = msg.StreamID
						c.writeEvent(fmsg)
						continue
					}
				}
				unableToHandle = true
			case types.MsgType_Error:
				if msg.Error != "" {
					return nil, fmt.Errorf("server error: %s", msg.Error)
				}
			}
			if unableToHandle {
				writeErr := c.writeEvent(types.Message{
					Type:     types.MsgType_StreamEnd,
					StreamID: msg.StreamID,
				})
				if writeErr != nil {
					c.logger.Log(ctx, types.LogType_Error, "failed to write stream end: %v\n", writeErr)
				}
				continue
			}
		}

		// Check for conversation end conditions
		if msg.Type == types.MsgType_StreamEnd {
			break
		}
	}

	response.LastAssistantMsg = c.lastAssistantMsg
	return &response, nil
}

// handleSingleToolCallback handles a single tool callback request using the WebSocket stream protocol
func (c *serverSession) handleSingleToolCallback(ctx context.Context, toolCallRequest types.Message, toolCallback types.ToolCallback) {
	if c.stream == nil {
		return
	}
	streamID := toolCallRequest.StreamID
	if streamID == "" || toolCallRequest.ToolName == "" {
		return
	}

	// ack
	err := c.stream.ACK(streamID)
	if err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to ack stream: %v\n", err)
		return
	}

	argsJSON := toolCallRequest.Content
	var args map[string]interface{}
	if err := unmarshalSafe([]byte(argsJSON), &args); err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to unmarshal arguments: %v\n", err)
		return
	}

	var workingDir string
	if toolCallRequest.Metadata.StreamRequestTool != nil {
		workingDir = toolCallRequest.Metadata.StreamRequestTool.DefaultWorkingDir
	}

	// Create ToolCall struct
	call := types.ToolCall{
		ID:         streamID,
		Name:       toolCallRequest.ToolName,
		Arguments:  args,
		RawArgs:    argsJSON,
		WorkingDir: workingDir,
	}

	// Execute the tool callback
	result, handled, err := toolCallback(ctx, c.stream, call)

	var toolError string
	if err != nil {
		toolError = err.Error()
	}

	var contentAsStr string
	switch res := result.Content.(type) {
	case string:
		contentAsStr = res
	case []byte:
		contentAsStr = string(res)
	}

	var resultJSON string
	if contentAsStr != "" {
		if !strings.HasPrefix(contentAsStr, "{") || !strings.HasSuffix(contentAsStr, "}") {
			resultJSON = fmt.Sprintf("{\"result\": %q}", contentAsStr)
		} else {
			resultJSON = contentAsStr
		}
	} else {
		json, err := json.Marshal(result.Content)
		if err != nil {
			resultJSON = fmt.Sprintf("failed to marshal result: %v", result.Content)
		} else {
			resultJSON = string(json)
		}
	}

	response := types.Message{
		Type:     types.MsgType_StreamResponseTool,
		StreamID: streamID,
		ToolName: toolCallRequest.ToolName,
		Content:  resultJSON,
		Error:    toolError,
		Metadata: types.Metadata{
			StreamResponseTool: &types.StreamResponseToolMetadata{
				OK: handled,
			},
		},
	}

	err = c.writeEvent(response)
	if err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to write response: %v\n", err)
	}
}

// websocketStreamContext implements types.StreamContext for WebSocket connections
type websocketStreamContext struct {
	conn *websocket.Conn
}

func (w *websocketStreamContext) ACK(id string) error {
	return w.Write(types.Message{
		Type:     types.MsgType_StreamHandleAck,
		StreamID: id,
	})
}

func (w *websocketStreamContext) Write(msg types.Message) error {
	msg = msg.TimeFilled()
	return w.conn.WriteJSON(msg)
}
