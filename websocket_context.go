package signalr

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

// contextualWebSocketConn wraps gorilla/websocket.Conn with context support
type contextualWebSocketConn struct {
	*websocket.Conn
}

func (c *contextualWebSocketConn) WriteWithContext(ctx context.Context, messageType int, data []byte) error {
	type result struct {
		err error
	}

	resultChan := make(chan result, 1)
	go func() {
		err := c.WriteMessage(messageType, data)
		resultChan <- result{err}
	}()

	select {
	case <-ctx.Done():
		c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context cancelled"),
			time.Now().Add(time.Second))
		c.Close()
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
		c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context cancelled"),
			time.Now().Add(time.Second))
		c.Close()
		return 0, nil, ctx.Err()
	case r := <-resultChan:
		return r.messageType, r.data, r.err
	}
}

func (c *contextualWebSocketConn) CloseWithReason(code int, reason string) error {
	return c.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}

func wrapGorillaConn(conn *websocket.Conn) *contextualWebSocketConn {
	return &contextualWebSocketConn{Conn: conn}
}
