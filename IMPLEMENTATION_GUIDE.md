# WebSocket Migration Implementation Guide

This document provides detailed, file-by-file implementation guidance for migrating from `github.com/coder/websocket` to `github.com/gorilla/websocket`.

## Prerequisites

### 1. Update Dependencies

First, update the `go.mod` file:

```bash
# Remove old dependency and add new one
go mod edit -droprequire github.com/coder/websocket
go get github.com/gorilla/websocket@latest
go mod tidy
```

### 2. Create Context Helper Functions

Create a new file `websocket_context.go` to handle context-aware operations:

```go
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
        c.Close(websocket.CloseNormalClosure, "context cancelled")
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
        c.Close(websocket.CloseNormalClosure, "context cancelled")
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
```

## File-by-File Implementation

### 1. websocketconnection.go

**Current Implementation Analysis:**
- Uses `*websocket.Conn` from coder/websocket
- Context-aware Read/Write operations
- Message type constants

**Migration Steps:**

```go
package signalr

import (
    "bytes"
    "context"
    "fmt"
    "net/http"
    "net/url"
    "github.com/gorilla/websocket"  // CHANGED: import
)

// Update function signature to use gorilla dialer
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
```

### 2. httpconnection.go

**Current Implementation Analysis:**
- Line 175: `websocket.Dial(ctx, wsURL.String(), opts)`
- Uses `websocket.DialOptions`

**Migration Steps:**

```go
// CHANGED: Update import
import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "path"
    "time"

    "github.com/quic-go/webtransport-go"
    "github.com/gorilla/websocket"  // CHANGED: import
)

// Around line 153-181, replace the websocket connection logic:
case httpConn.hasTransport(TransportWebSockets) && negotiateResponse.hasTransport(TransportWebSockets):
    wsURL := reqURL

    // switch to wss for secure connection
    if reqURL.Scheme == "https" {
        wsURL.Scheme = "wss"
    } else {
        wsURL.Scheme = "ws"
    }

    // CHANGED: Use gorilla dialer instead of DialOptions
    dialer := &websocket.Dialer{
        HandshakeTimeout: time.Second * 30,
    }

    headers := http.Header{}
    if httpConn.headers != nil {
        headers = httpConn.headers()
    }

    for _, cookie := range resp.Cookies() {
        headers.Add("Cookie", cookie.String())
    }

    // CHANGED: Use dialer.DialContext instead of websocket.Dial
    ws, _, err := dialer.DialContext(ctx, wsURL.String(), headers)
    if err != nil {
        return nil, err
    }

    // TODO think about if the API should give the possibility to cancel this connection
    conn = newWebSocketConnection(context.Background(), negotiateResponse.ConnectionID, ws)
```

### 3. httpmux.go

**Current Implementation Analysis:**
- Line 147: `websocket.AcceptOptions`
- Line 152: `websocket.Accept(writer, request, accOptions)`

**Migration Steps:**

```go
// CHANGED: Update import
import (
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/teivah/onecontext"
    "github.com/gorilla/websocket"  // CHANGED: import
)

// Around line 146-159, replace the websocket accept logic:
func (h *httpMux) handleWebsocket(writer http.ResponseWriter, request *http.Request) {
    // CHANGED: Use gorilla Upgrader instead of AcceptOptions
    upgrader := &websocket.Upgrader{
        ReadBufferSize:    1024,
        WriteBufferSize:   1024,
        EnableCompression: true,  // CHANGED: equivalent to CompressionContextTakeover
        CheckOrigin: func(r *http.Request) bool {
            // CHANGED: Implement origin checking based on server settings
            if h.server.insecureSkipVerify() {
                return true
            }
            
            origin := r.Header.Get("Origin")
            if origin == "" {
                return true
            }
            
            patterns := h.server.originPatterns()
            if len(patterns) == 0 {
                return true
            }
            
            for _, pattern := range patterns {
                if pattern == "*" || pattern == origin {
                    return true
                }
                // Add more sophisticated pattern matching if needed
            }
            return false
        },
    }
    
    // CHANGED: Use upgrader.Upgrade instead of websocket.Accept
    websocketConn, err := upgrader.Upgrade(writer, request, nil)
    if err != nil {
        _, debug := h.server.loggers()
        _ = debug.Log(evt, "handleWebsocket", msg, "error upgrading websockets", "error", err)
        // Note: upgrader.Upgrade handles HTTP error responses automatically
        return
    }
    
    // CHANGED: Set read limit using gorilla API
    websocketConn.SetReadLimit(int64(h.server.maximumReceiveMessageSize()))
    
    connectionMapKey := request.URL.Query().Get("id")
    if connectionMapKey == "" {
        // Support websocket connection without negotiate
        connectionMapKey = newConnectionID()
        h.mx.Lock()
        h.connectionMap[connectionMapKey] = &negotiateConnection{
            ConnectionBase{connectionID: connectionMapKey},
        }
        h.mx.Unlock()
    }
    
    h.mx.RLock()
    c, ok := h.connectionMap[connectionMapKey]
    h.mx.RUnlock()
    if ok {
        if _, ok := c.(*negotiateConnection); ok {
            // Connection is negotiated but not initiated
            ctx, _ := onecontext.Merge(h.server.Context(), request.Context())
            err = h.serveConnection(newWebSocketConnection(ctx, c.ConnectionID(), websocketConn))
            if err != nil {
                // CHANGED: Use gorilla close method
                _ = websocketConn.WriteControl(websocket.CloseMessage,
                    websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()),
                    time.Now().Add(time.Second))
                _ = websocketConn.Close()
            }
        } else {
            // Already initiated
            _ = websocketConn.WriteControl(websocket.CloseMessage,
                websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Bad request"),
                time.Now().Add(time.Second))
            _ = websocketConn.Close()
        }
    } else {
        // Not negotiated
        _ = websocketConn.WriteControl(websocket.CloseMessage,
            websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Not found"),
            time.Now().Add(time.Second))
        _ = websocketConn.Close()
    }
}
```

### 4. httpserver_test.go

**Current Implementation Analysis:**
- Line 21: Import statement
- Test code that uses websocket client

**Migration Steps:**

```go
// CHANGED: Update import
import (
    // ... other imports
    "github.com/gorilla/websocket"  // CHANGED: import
)

// Find test functions that create websocket connections and update them:
// Example of updating test websocket client code:

func createTestWebSocketClient(url string) (*websocket.Conn, error) {
    // CHANGED: Use gorilla dialer
    dialer := &websocket.Dialer{
        HandshakeTimeout: time.Second * 10,
    }
    
    conn, _, err := dialer.Dial(url, nil)
    return conn, err
}

// Update any test code that uses websocket message types:
// websocket.MessageText -> websocket.TextMessage
// websocket.MessageBinary -> websocket.BinaryMessage

// Update any test code that reads/writes messages:
// conn.Write(ctx, messageType, data) -> conn.WriteMessage(messageType, data)
// messageType, data, err := conn.Read(ctx) -> messageType, data, err := conn.ReadMessage()
```

### 5. serveroptions.go

**Current Implementation Analysis:**
- Lines 63, 72: Documentation references to coder/websocket

**Migration Steps:**

```go
// CHANGED: Update documentation URLs and comments

// InsecureSkipVerify disables origin verification behaviour which is used to avoid same origin strategy.
// See https://pkg.go.dev/github.com/gorilla/websocket#Upgrader
func InsecureSkipVerify(skip bool) func(Party) error {
    return func(p Party) error {
        p.setInsecureSkipVerify(skip)
        return nil
    }
}

// AllowOriginPatterns lists the host patterns for authorized origins which is used for avoid same origin strategy.
// See https://pkg.go.dev/github.com/gorilla/websocket#Upgrader
func AllowOriginPatterns(origins []string) func(Party) error {
    return func(p Party) error {
        p.setOriginPatterns(origins)
        return nil
    }
}
```

## Post-Migration Testing

### 1. Unit Tests
Run specific websocket-related tests:
```bash
go test -v -run WebSocket
go test -v -run websocket
```

### 2. Integration Tests
Test the full SignalR flow:
```bash
go test -v ./...
```

### 3. Manual Testing
1. Start the chat sample: `cd chatsample && go run main.go`
2. Open multiple browser tabs to test concurrent connections
3. Test text and binary message sending
4. Test connection cleanup on browser close

### 4. Performance Testing
```bash
go test -bench=. -benchmem
```

## Validation Checklist

- [ ] All imports updated to gorilla/websocket
- [ ] Message type constants updated (MessageText -> TextMessage, etc.)
- [ ] Dial operations use Dialer struct
- [ ] Accept operations use Upgrader struct
- [ ] Context handling implemented for read/write operations
- [ ] Origin checking logic migrated properly
- [ ] Compression settings migrated
- [ ] All tests pass
- [ ] Documentation URLs updated
- [ ] No remaining references to coder/websocket

## Troubleshooting Common Issues

### 1. Context Cancellation Not Working
**Problem:** Gorilla websocket doesn't support context natively
**Solution:** Use the wrapper functions in `websocket_context.go`

### 2. Origin Check Failures
**Problem:** Different origin checking mechanism
**Solution:** Implement CheckOrigin function in Upgrader

### 3. Message Type Errors
**Problem:** Different constant values between libraries
**Solution:** Update all references:
- `websocket.MessageText` → `websocket.TextMessage`
- `websocket.MessageBinary` → `websocket.BinaryMessage`

### 4. Compression Issues
**Problem:** Different compression configuration
**Solution:** Use `EnableCompression: true` in Upgrader

### 5. Close Code Issues
**Problem:** Different close code handling
**Solution:** Use `websocket.FormatCloseMessage()` and `WriteControl()`
