package signalr

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// contextualWebSocketConn wraps gorilla/websocket.Conn with context support
type contextualWebSocketConn struct {
	*websocket.Conn
	writeMu sync.Mutex // Mutex to prevent concurrent writes
}

func (c *contextualWebSocketConn) WriteWithContext(ctx context.Context, messageType int, data []byte) error {
	type result struct {
		err error
	}

	resultChan := make(chan result, 1)
	go func() {
		// Acquire write mutex to prevent concurrent writes to the same WebSocket connection
		c.writeMu.Lock()
		err := c.WriteMessage(messageType, data)
		c.writeMu.Unlock()
		resultChan <- result{err}
	}()

	select {
	case <-ctx.Done():
		// Note: WriteControl operations also need synchronization
		c.writeMu.Lock()
		c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context cancelled"),
			time.Now().Add(time.Second))
		c.Close()
		c.writeMu.Unlock()
		return ctx.Err()
	case r := <-resultChan:
		return r.err
	}
}

func (c *contextualWebSocketConn) ReadWithContext(ctx context.Context) (int, []byte, error) {
	type result struct {
		messageType int
		data        []byte
		err         error
	}

	resultChan := make(chan result, 1)
	go func() {
		messageType, data, err := c.ReadMessage()
		resultChan <- result{messageType, data, err}
	}()

	select {
	case <-ctx.Done():
		// Synchronize WriteControl operations to prevent race conditions
		c.writeMu.Lock()
		c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context cancelled"),
			time.Now().Add(time.Second))
		c.Close()
		c.writeMu.Unlock()
		return 0, nil, ctx.Err()
	case r := <-resultChan:
		return r.messageType, r.data, r.err
	}
}

func (c *contextualWebSocketConn) CloseWithReason(code int, reason string) error {
	// Synchronize WriteControl operations to prevent race conditions
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}

func wrapGorillaConn(conn *websocket.Conn) *contextualWebSocketConn {
	return &contextualWebSocketConn{Conn: conn}
}
