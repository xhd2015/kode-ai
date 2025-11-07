package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/kode-ai/types/providers"
	"github.com/xhd2015/kode-ai/types"
)

// serverSession handles WebSocket server communication
type serverSession struct {
	stream        types.StreamContext
	eventCallback types.EventCallback

	eventBuf chan types.Message

	logger types.Logger

	lastAssistantMsg string
}

// ChatWithServer connects to a WebSocket chat server and streams events until finished
func ChatWithServer(ctx context.Context, server string, req types.Request) (*types.Response, error) {
	sess := &serverSession{
		eventCallback: req.EventCallback,
		logger:        getLogger(req.Logger),
		eventBuf:      make(chan types.Message, 10),
	}
	return sess.chatWithServer(ctx, server, req)
}

// chatWithServer connects to a WebSocket server and handles the streaming protocol
func (c *serverSession) chatWithServer(ctx context.Context, server string, req types.Request) (*types.Response, error) {
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

	// Connect to WebSocket with handshake timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer conn.Close()

	// Set up ping/pong handler for connection health
	conn.SetPongHandler(func(string) error {
		return nil
	})

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
	return c.processWebSocketMessages(ctx, conn, req.Model, req.ToolCallback, req.FollowUpCallback, req.ToolDefinitions)
}

// writeEvent writes an event and calls the event callback
func (c *serverSession) writeEventBuf(msg types.Message) {
	c.eventBuf <- msg
}

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
func (c *serverSession) processWebSocketMessages(ctx context.Context, conn *websocket.Conn, model string, toolCallback types.ToolCallback, followUpCallback types.FollowUpCallback, toolDefs []*types.UnifiedTool) (*types.Response, error) {
	var response types.Response

	// ping every 10s
	pingTicker := time.NewTicker(10 * time.Second)
	defer pingTicker.Stop()

	msgChan := make(chan types.Message)
	errChan := make(chan error)
	done := make(chan struct{})
	defer close(done)

	go func() {
		defer close(msgChan)
		defer close(errChan)
		for {
			var msg types.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				select {
				case errChan <- err:
				case <-done:
				}
				return
			}
			select {
			case msgChan <- msg:
			case <-done:
				return
			}
		}
	}()

	for {
		var msg types.Message
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg := <-c.eventBuf:
			err := c.writeEvent(msg)
			if err != nil {
				c.logger.Log(ctx, types.LogType_Error, "failed to write event: %v\n", err)
				return nil, fmt.Errorf("failed to write event: %w", err)
			}
			continue
		case <-pingTicker.C:
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				c.logger.Log(ctx, types.LogType_Error, "failed to ping: %v\n", err)
			}
			continue
		case err := <-errChan:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break // Normal close
			}
			return nil, fmt.Errorf("failed to read WebSocket message: %w", err)
		case msg = <-msgChan:
			// handled below
		default:
			continue
		}

		if msg.Type == types.MsgType_ToolCall {
			response.NumToolCalls++
		}

		if msg.Type == types.MsgType_Msg && msg.Role == types.Role_Assistant {
			c.lastAssistantMsg = msg.Content
		}
		if msg.Type == types.MsgType_TokenUsage && msg.TokenUsage != nil {
			tokenUsage := msg.TokenUsage
			if msg.TokenCost == nil {
				response.TokenUsage = response.TokenUsage.Add(*tokenUsage)
				provider, _ := providers.GetModelAPIShape(model)
				if provider != "" {
					modelCost, ok := providers.ComputeCost(provider, model, *msg.TokenUsage)
					if ok {
						msg.TokenCost = &modelCost
					}
				}
			}
			if msg.TokenCost != nil {
				if response.Cost == nil {
					response.Cost = &types.TokenCost{}
				}
				*response.Cost = response.Cost.Add(*msg.TokenCost)
			}
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
					err := c.stream.ACK(msg.StreamID)
					if err != nil {
						c.logger.Log(ctx, types.LogType_Error, "failed to ack stream: %v\n", err)
						continue
					}
					// start a new goroutine to handle the tool callback
					// so that it won't block the main loop
					go c.handleSingleToolCallbackAsync(ctx, msg.StreamID, msg, foundToolCallback)
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
					go func() {
						followUpResult, err := followUpCallback(ctx)
						if err != nil {
							c.writeEventBuf(types.Message{
								Type:     types.MsgType_Error,
								StreamID: msg.StreamID,
								Error:    err.Error(),
							})
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
							c.writeEventBuf(fmsg)
						} else {
							c.writeEventBuf(types.Message{
								Type:     types.MsgType_StreamEnd,
								StreamID: msg.StreamID,
							})
						}
					}()
					continue
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

// handleSingleToolCallbackAsync handles a single tool callback request using the WebSocket stream protocol
func (c *serverSession) handleSingleToolCallbackAsync(ctx context.Context, streamID string, toolCallRequest types.Message, toolCallback types.ToolCallback) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Log(ctx, types.LogType_Error, "panic in handleSingleToolCallbackAsync: %v\n", r)
			response := types.Message{
				Type:     types.MsgType_StreamResponseTool,
				StreamID: streamID,
				ToolName: toolCallRequest.ToolName,
				Content:  "panic in handleSingleToolCallbackAsync",
				Error:    fmt.Sprintf("%v", r),
				Metadata: types.Metadata{
					StreamResponseTool: &types.StreamResponseToolMetadata{
						OK: true,
					},
				},
			}
			c.writeEventBuf(response)
		}
	}()

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
	c.writeEventBuf(response)
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
