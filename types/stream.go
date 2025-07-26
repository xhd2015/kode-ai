package types

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// StdinReader interface for background stdin reading
type StdinReader interface {
	Subscribe(id string) chan Message
	Unsubscribe(id string)
	Start()
}

// NewStdinReader creates a new stdin reader instance
func NewStdinReader(stdin io.Reader) StdinReader {
	return &stdinReaderImpl{
		stdin:    stdin,
		channels: make(map[string]chan Message),
	}
}

// stdinReaderImpl implements the StdinReader interface
type stdinReaderImpl struct {
	stdin    io.Reader
	channels map[string]chan Message
	once     sync.Once

	mutex sync.RWMutex
}

// Start begins the background reading loop (should be called once)
func (sr *stdinReaderImpl) Start() {
	sr.once.Do(func() {
		go sr.readLoop()
	})
}

// readLoop continuously reads from stdin and distributes messages
func (sr *stdinReaderImpl) readLoop() {
	scanner := bufio.NewScanner(sr.stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := unmarshalSafe([]byte(line), &msg); err != nil {
			continue // Skip non-JSON lines
		}

		if msg.StreamID == "" {
			continue // Skip messages without ID
		}

		// Distribute message to the appropriate channel
		sr.mutex.RLock()
		ch, exists := sr.channels[msg.StreamID]
		sr.mutex.RUnlock()

		if exists {
			ch <- msg
		}
	}
}

// Subscribe creates a channel for receiving messages with the given ID
func (sr *stdinReaderImpl) Subscribe(id string) chan Message {
	// ensure started
	sr.Start()

	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	ch := make(chan Message, 10) // Buffered channel
	sr.channels[id] = ch
	return ch
}

// Unsubscribe removes the channel for the given ID
func (sr *stdinReaderImpl) Unsubscribe(id string) {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if ch, exists := sr.channels[id]; exists {
		close(ch)
		delete(sr.channels, id)
	}
}

// const STREAM_ACK_TIMEOUT = 100 * time.Second
const STREAM_ACK_TIMEOUT = 1 * time.Second

var ErrStreamEnd = fmt.Errorf("stream end")

// if expectMsgType is empty, it will return the first message that is not a stream handle ack
func StreamRequest(ctx context.Context, writer io.Writer, reader StdinReader, requestMsg Message, expectMsgType MsgType) (Message, error) {
	if requestMsg.StreamID == "" {
		// new random uuid
		return Message{}, fmt.Errorf("requires stream id")
	}
	requestMsg = requestMsg.TimeFilled()

	id := requestMsg.StreamID

	// Subscribe to messages for this tool call ID
	msgChan := reader.Subscribe(id)
	defer reader.Unsubscribe(id)

	// Output tool call request to stdout as JSON
	requestJSON, err := json.Marshal(requestMsg)
	if err != nil {
		return Message{}, fmt.Errorf("marshal request: %w", err)
	}

	if _, err := fmt.Fprintln(writer, string(requestJSON)); err != nil {
		return Message{}, fmt.Errorf("write request: %w", err)
	}

	// Create a timeout context for the acknowledgment phase (1 second)
	ackCtx, ackCancel := context.WithTimeout(ctx, STREAM_ACK_TIMEOUT)
	defer ackCancel()

	// Wait for acknowledgment or timeout
	var ackReceived bool
	for !ackReceived {
		select {
		case msg := <-msgChan:
			if msg.Type == MsgType_StreamHandleAck {
				ackReceived = true
			} else if msg.Type == MsgType_Error {
				return msg, fmt.Errorf("error: %s", msg.Error)
			} else if msg.Type == MsgType_StreamEnd {
				return msg, ErrStreamEnd
			} else if expectMsgType == "" || msg.Type == expectMsgType {
				return msg, nil
			}
		case <-ackCtx.Done():
			return Message{}, fmt.Errorf("timeout waiting for acknowledgment (1s)")
		}
	}

	// Now wait for the actual tool response (no timeout for execution)
	for {
		select {
		case msg := <-msgChan:
			if msg.Type == MsgType_Error {
				return msg, fmt.Errorf("error: %s", msg.Error)
			} else if msg.Type == MsgType_StreamEnd {
				return msg, ErrStreamEnd
			}
			if expectMsgType == "" || msg.Type == expectMsgType {
				return msg, nil
			}
		case <-ctx.Done():
			return Message{}, ctx.Err()
		}
	}
}
