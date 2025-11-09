package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/kode-ai/chat"
	"github.com/xhd2015/kode-ai/types"
)

// ServerOptions represents the configuration options for the chat server
type ServerOptions struct {
	Verbose bool // Enable verbose logging
}

// Server represents the chat server
type Server struct {
	port   int
	opts   ServerOptions
	server *http.Server
}

// NewServer creates a new chat server
func NewServer(port int, opts ServerOptions) (*Server, error) {
	server := &Server{
		port: port,
		opts: opts,
	}
	return server, nil
}

// Start starts the HTTP server
func Start(port int, opts ServerOptions) error {
	server, err := NewServer(port, opts)
	if err != nil {
		return err
	}
	return server.Start()
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleWebSocket)
	mux.HandleFunc("/shutdown", s.handleShutdown)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting chat server on %s", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s.server = server

	err := server.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server shutdown gracefully")
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.opts.Verbose {
		log.Printf("WebSocket connection request from %s", r.RemoteAddr)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer func() {
		if s.opts.Verbose {
			log.Printf("Closing WebSocket connection from %s", r.RemoteAddr)
		}
		conn.Close()
	}()

	if s.opts.Verbose {
		log.Printf("WebSocket connection established with %s", r.RemoteAddr)
	}

	// Check for wait_for_stream_events query parameter
	waitForStreamEvents := false
	for key, values := range r.URL.Query() {
		if key == "wait_for_stream_events" {
			if len(values) == 0 || values[0] == "true" {
				waitForStreamEvents = true
			}
			break
		}
	}
	msg := r.URL.Query().Get("message")
	baseURL := r.URL.Query().Get("base_url")
	model := r.URL.Query().Get("model")
	token := r.URL.Query().Get("token")
	systemPrompt := r.URL.Query().Get("system_prompt")

	if s.opts.Verbose {
		log.Printf("Request parameters: waitForStreamEvents=%t, model=%s, baseURL=%s, hasToken=%t, messageLen=%d, systemPromptLen=%d",
			waitForStreamEvents, model, baseURL, token != "", len(msg), len(systemPrompt))
	}

	ctx := context.Background()

	// Create WebSocket-based stream reader
	wsReader := NewWebSocketReader(conn)
	wsReader.verbose = s.opts.Verbose
	wsReader.Start()
	defer wsReader.Close()

	if s.opts.Verbose {
		log.Printf("WebSocket reader started for %s", r.RemoteAddr)
	}

	// Add WebSocket stream support
	req := types.Request{
		Message:      msg,
		Model:        model,
		Token:        token,
		BaseURL:      baseURL,
		SystemPrompt: systemPrompt,
	}

	// If waiting for stream events, load initial events
	if waitForStreamEvents {
		if s.opts.Verbose {
			log.Printf("Loading initial events from WebSocket for %s", r.RemoteAddr)
		}
		messages, err := s.loadInitialEventsFromWebSocket(ctx, wsReader, &req, 30*time.Second)
		if err != nil {
			log.Printf("Failed to load initial events: %v", err)
			s.sendError(conn, fmt.Sprintf("Failed to load initial events: %v", err))
			return
		}
		req.History = s.convertMessagesToHistory(messages)
		if s.opts.Verbose {
			log.Printf("Loaded %d initial events, converted to %d history messages for %s", len(messages), len(req.History), r.RemoteAddr)
		}
	}

	req.StreamPair = &types.StreamPair{
		Input:  wsReader,
		Output: NewWebSocketWriter(conn, s.opts.Verbose),
	}
	req.EventCallback = func(event types.Message) {
		event = event.TimeFilled()
		if s.opts.Verbose {
			log.Printf("Sending event to %s: type=%s, role=%s, contentLen=%d", r.RemoteAddr, event.Type, event.Role, len(event.Content))
		}
		if err := conn.WriteJSON(event); err != nil {
			log.Printf("Failed to send event: %v", err)
		}
	}

	if s.opts.Verbose {
		log.Printf("Starting chat execution for %s", r.RemoteAddr)
	}

	// Execute chat
	_, err = chat.Chat(ctx, req)
	if err != nil {
		log.Printf("Chat execution failed: %v", err)
		s.sendError(conn, fmt.Sprintf("Chat execution failed: %v", err))
		return
	}

	if s.opts.Verbose {
		log.Printf("Chat execution completed successfully for %s", r.RemoteAddr)
	}

	// Send stream end event to signal completion
	endEvent := types.Message{
		Type: types.MsgType_StreamEnd,
	}.TimeFilled()

	if s.opts.Verbose {
		log.Printf("Sending stream end event to %s", r.RemoteAddr)
	}

	if err := conn.WriteJSON(endEvent); err != nil {
		log.Printf("Failed to send stream end event: %v", err)
	}

	if s.opts.Verbose {
		log.Printf("Sending close message to %s", r.RemoteAddr)
	}

	// Close the connection gracefully
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if s.opts.Verbose {
		log.Printf("Shutdown request received from %s", r.RemoteAddr)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Shutting down...\n"))
	go func() {
		time.Sleep(200 * time.Millisecond)
		if s.opts.Verbose {
			log.Printf("Initiating server shutdown")
		}
		s.Shutdown(r.Context())
	}()
}

func (s *Server) sendError(conn *websocket.Conn, errorMsg string) {
	if s.opts.Verbose {
		log.Printf("Sending error message: %s", errorMsg)
	}

	errorEvent := types.Message{
		Type:    types.MsgType_Error,
		Content: errorMsg,
		Error:   errorMsg,
	}.TimeFilled()

	if err := conn.WriteJSON(errorEvent); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

func (s *Server) loadInitialEventsFromWebSocket(ctx context.Context, reader *WebSocketReader, req *types.Request, timeout time.Duration) ([]types.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var messages []types.Message

	if s.opts.Verbose {
		log.Printf("Loading initial events with timeout %v", timeout)
	}

	for {
		select {
		case msg := <-reader.MessageChan():
			if s.opts.Verbose {
				log.Printf("Received initial event: type=%s, contentLen=%d", msg.Type, len(msg.Content))
			}
			switch msg.Type {
			case types.MsgType_StreamInitRequest:
				var initReq types.Request
				if err := json.Unmarshal([]byte(msg.Content), &initReq); err != nil {
					return nil, fmt.Errorf("failed to unmarshal init request: %v", err)
				}
				*req = initReq
				if s.opts.Verbose {
					log.Printf("Processed stream init request: model=%s, maxRounds=%d, toolsCount=%d", req.Model, req.MaxRounds, len(req.ToolDefinitions))
				}
			case types.MsgType_StreamInitEventsFinished:
				if s.opts.Verbose {
					log.Printf("Initial events loading finished, collected %d messages", len(messages))
				}
				return messages, nil
			default:
				messages = append(messages, msg)
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for initial events")
		}
	}
}

func (s *Server) convertMessagesToHistory(messages []types.Message) []types.Message {
	var history []types.Message
	for _, msg := range messages {
		if msg.Type.HistorySendable() {
			history = append(history, msg)
		}
	}
	if s.opts.Verbose {
		log.Printf("Converted %d messages to %d history messages", len(messages), len(history))
	}
	return history
}

// WebSocketReader implements types.StdinReader for WebSocket connections
type WebSocketReader struct {
	conn     *websocket.Conn
	channels map[string]chan types.Message
	msgChan  chan types.Message
	done     chan struct{}
	mutex    sync.RWMutex
	verbose  bool
}

// NewWebSocketReader creates a new WebSocket reader
func NewWebSocketReader(conn *websocket.Conn) *WebSocketReader {
	return &WebSocketReader{
		conn:     conn,
		channels: make(map[string]chan types.Message),
		msgChan:  make(chan types.Message, 100),
		done:     make(chan struct{}),
	}
}

func (wr *WebSocketReader) Start() {
	go wr.readLoop()
}

func (wr *WebSocketReader) Close() {
	close(wr.done)
}

func (wr *WebSocketReader) MessageChan() <-chan types.Message {
	return wr.msgChan
}

func (wr *WebSocketReader) readLoop() {
	if wr.verbose {
		log.Printf("WebSocket reader loop started")
	}
	for {
		select {
		case <-wr.done:
			if wr.verbose {
				log.Printf("WebSocket reader loop stopping")
			}
			return
		default:
			var msg types.Message
			err := wr.conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				} else if wr.verbose {
					log.Printf("WebSocket read ended: %v", err)
				}
				return
			}

			if wr.verbose {
				log.Printf("WebSocket received message: type=%s, streamID=%s, contentLen=%d", msg.Type, msg.StreamID, len(msg.Content))
			}

			// Send to general message channel
			select {
			case wr.msgChan <- msg:
			case <-wr.done:
				return
			}

			// Also send to specific stream ID channels if applicable
			if msg.StreamID != "" {
				wr.mutex.RLock()
				ch, exists := wr.channels[msg.StreamID]
				wr.mutex.RUnlock()

				if exists {
					if wr.verbose {
						log.Printf("Forwarding message to stream channel: streamID=%s", msg.StreamID)
					}
					select {
					case ch <- msg:
					case <-wr.done:
						return
					}
				} else if wr.verbose {
					log.Printf("No channel found for streamID: %s", msg.StreamID)
				}
			}
		}
	}
}

func (wr *WebSocketReader) Subscribe(id string) chan types.Message {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	ch := make(chan types.Message, 10)
	wr.channels[id] = ch
	if wr.verbose {
		log.Printf("Subscribed to stream channel: streamID=%s", id)
	}
	return ch
}

func (wr *WebSocketReader) Unsubscribe(id string) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	// if there are pending writers, we should notify them
	// that the channel is closed
	if _, exists := wr.channels[id]; exists {
		delete(wr.channels, id)
		if wr.verbose {
			log.Printf("Unsubscribed from stream channel: streamID=%s", id)
		}
	} else if wr.verbose {
		log.Printf("Attempted to unsubscribe from non-existent stream channel: streamID=%s", id)
	}
}

// Read implements io.Reader interface
func (wr *WebSocketReader) Read(p []byte) (n int, err error) {
	select {
	case msg := <-wr.msgChan:
		data, err := json.Marshal(msg)
		if err != nil {
			return 0, err
		}
		data = append(data, '\n') // Add newline to match stdin behavior

		if len(data) > len(p) {
			return 0, fmt.Errorf("buffer too small")
		}

		copy(p, data)
		return len(data), nil
	case <-wr.done:
		return 0, io.EOF
	}
}

// WebSocketWriter implements io.Writer for WebSocket connections
type WebSocketWriter struct {
	conn    *websocket.Conn
	verbose bool
}

func NewWebSocketWriter(conn *websocket.Conn, verbose bool) *WebSocketWriter {
	return &WebSocketWriter{conn: conn, verbose: verbose}
}

func (ww *WebSocketWriter) Write(p []byte) (n int, err error) {
	if ww.verbose {
		log.Printf("WebSocket writer sending %d bytes", len(p))
	}
	err = ww.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		if ww.verbose {
			log.Printf("WebSocket write failed: %v", err)
		}
		return 0, err
	}
	return len(p), nil
}
