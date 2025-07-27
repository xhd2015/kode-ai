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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

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

	ctx := context.Background()

	// Create WebSocket-based stream reader
	wsReader := NewWebSocketReader(conn)
	wsReader.Start()
	defer wsReader.Close()

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
		messages, err := s.loadInitialEventsFromWebSocket(ctx, wsReader, &req, 30*time.Second)
		if err != nil {
			log.Printf("Failed to load initial events: %v", err)
			s.sendError(conn, fmt.Sprintf("Failed to load initial events: %v", err))
			return
		}
		req.History = s.convertMessagesToHistory(messages)
	}

	req.StreamPair = &types.StreamPair{
		Input:  wsReader,
		Output: NewWebSocketWriter(conn),
	}
	req.EventCallback = func(event types.Message) {
		event = event.TimeFilled()
		if err := conn.WriteJSON(event); err != nil {
			log.Printf("Failed to send event: %v", err)
		}
	}

	// Execute chat
	_, err = chat.Chat(ctx, req)
	if err != nil {
		log.Printf("Chat execution failed: %v", err)
		s.sendError(conn, fmt.Sprintf("Chat execution failed: %v", err))
		return
	}

	// Send stream end event to signal completion
	endEvent := types.Message{
		Type: types.MsgType_StreamEnd,
	}.TimeFilled()

	if err := conn.WriteJSON(endEvent); err != nil {
		log.Printf("Failed to send stream end event: %v", err)
	}

	// Close the connection gracefully
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Shutting down...\n"))
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.Shutdown(r.Context())
	}()
}

func (s *Server) sendError(conn *websocket.Conn, errorMsg string) {
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

	for {
		select {
		case msg := <-reader.MessageChan():
			switch msg.Type {
			case types.MsgType_StreamInitRequest:
				var initReq types.Request
				if err := json.Unmarshal([]byte(msg.Content), &initReq); err != nil {
					return nil, fmt.Errorf("failed to unmarshal init request: %v", err)
				}
				*req = initReq
			case types.MsgType_StreamInitEventsFinished:
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
	return history
}

// WebSocketReader implements types.StdinReader for WebSocket connections
type WebSocketReader struct {
	conn     *websocket.Conn
	channels map[string]chan types.Message
	msgChan  chan types.Message
	done     chan struct{}
	mutex    sync.RWMutex
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
	for {
		select {
		case <-wr.done:
			return
		default:
			var msg types.Message
			err := wr.conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
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
					select {
					case ch <- msg:
					case <-wr.done:
						return
					}
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
	return ch
}

func (wr *WebSocketReader) Unsubscribe(id string) {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	if ch, exists := wr.channels[id]; exists {
		close(ch)
		delete(wr.channels, id)
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
	conn *websocket.Conn
}

func NewWebSocketWriter(conn *websocket.Conn) *WebSocketWriter {
	return &WebSocketWriter{conn: conn}
}

func (ww *WebSocketWriter) Write(p []byte) (n int, err error) {
	err = ww.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
