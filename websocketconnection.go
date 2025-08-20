package signalr

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"github.com/gorilla/websocket"
)

func NewWebSocketConnection(ctx context.Context, reqURL *url.URL, connectionID string, headers http.Header) (Connection, error) {
	// CHANGED: Use gorilla dialer instead of direct Dial
	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second * 30,
	}
	
	ws, _, err := dialer.Dial(reqURL.String(), headers)  // CHANGED: API
	if err != nil {
		return nil, err
	}

	return newWebSocketConnection(ctx, connectionID, ws), nil
}

type webSocketConnection struct {
	ConnectionBase
	conn         *contextualWebSocketConn  // CHANGED: Use wrapped conn
	transferMode TransferMode
}

func newWebSocketConnection(ctx context.Context, connectionID string, conn *websocket.Conn) *webSocketConnection {
	w := &webSocketConnection{
		conn:           wrapGorillaConn(conn),  // CHANGED: Wrap conn
		ConnectionBase: *NewConnectionBase(ctx, connectionID),
	}
	return w
}

func (w *webSocketConnection) Write(p []byte) (n int, err error) {
	// CHANGED: Update message type constants
	messageType := websocket.TextMessage
	if w.transferMode == BinaryTransferMode {
		messageType = websocket.BinaryMessage
	}
	
	n, err = ReadWriteWithContext(w.Context(),
		func() (int, error) {
			// CHANGED: Use wrapped WriteWithContext
			err := w.conn.WriteWithContext(w.Context(), messageType, p)
			if err != nil {
				return 0, err
			}
			return len(p), nil
		},
		func() {})
	if err != nil {
		err = fmt.Errorf("%T: %w", w, err)
		// CHANGED: Use wrapped CloseWithReason
		_ = w.conn.CloseWithReason(websocket.CloseNormalClosure, err.Error())
	}
	return n, err
}

func (w *webSocketConnection) Read(p []byte) (n int, err error) {
	n, err = ReadWriteWithContext(w.Context(),
		func() (int, error) {
			// CHANGED: Use wrapped ReadWithContext
			_, data, err := w.conn.ReadWithContext(w.Context())
			if err != nil {
				return 0, err
			}
			return bytes.NewReader(data).Read(p)
		},
		func() {})
	if err != nil {
		err = fmt.Errorf("%T: %w", w, err)
		// CHANGED: Use wrapped CloseWithReason  
		_ = w.conn.CloseWithReason(websocket.CloseNormalClosure, err.Error())
	}
	return n, err
}

// TransferMode and SetTransferMode remain unchanged
func (w *webSocketConnection) TransferMode() TransferMode {
	return w.transferMode
}

func (w *webSocketConnection) SetTransferMode(transferMode TransferMode) {
	w.transferMode = transferMode
}
