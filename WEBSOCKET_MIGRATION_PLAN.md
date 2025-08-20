# WebSocket Library Migration Plan

## Objective
Replace `github.com/coder/websocket` with `github.com/gorilla/websocket` in the go-signalr library.

## Current State Analysis

### Dependencies
- Current: `github.com/coder/websocket v1.8.13` (in go.mod line 7)
- Target: `github.com/gorilla/websocket` (latest stable version)

### Affected Files
1. **httpconnection.go** (line 13) - Client-side websocket dialing
2. **websocketconnection.go** (line 9) - Core websocket connection implementation  
3. **httpmux.go** (line 15) - Server-side websocket accept/upgrade
4. **httpserver_test.go** (line 21) - Test implementations
5. **serveroptions.go** (lines 63, 72) - Documentation references

## API Differences Analysis

### Connection Establishment

#### Current (coder/websocket)
```go
// Client-side dialing
ws, _, err := websocket.Dial(ctx, wsURL.String(), opts)

// Server-side accept
websocketConn, err := websocket.Accept(writer, request, accOptions)
```

#### Target (gorilla/websocket)
```go
// Client-side dialing
dialer := websocket.Dialer{}
ws, _, err := dialer.Dial(wsURL.String(), headers)

// Server-side upgrade
upgrader := websocket.Upgrader{}
websocketConn, err := upgrader.Upgrade(writer, request, headers)
```

### Message Handling

#### Current (coder/websocket)
```go
// Context-based operations
err := conn.Write(ctx, messageType, data)
messageType, data, err := conn.Read(ctx)
err := conn.Close(code, reason)
```

#### Target (gorilla/websocket)
```go
// Traditional operations (context handling needs to be added manually)
err := conn.WriteMessage(messageType, data)
messageType, data, err := conn.ReadMessage()
err := conn.Close(code, reason)
```

### Configuration Options

#### Current (coder/websocket)
```go
// Dial options
opts := &websocket.DialOptions{
    HTTPHeader: headers,
}

// Accept options  
accOptions := &websocket.AcceptOptions{
    CompressionMode: websocket.CompressionContextTakeover,
    InsecureSkipVerify: true,
    OriginPatterns: []string{"*"},
}
```

#### Target (gorilla/websocket)
```go
// Dialer configuration
dialer := &websocket.Dialer{
    HandshakeTimeout: time.Second * 30,
    // Context handling needs manual implementation
}

// Upgrader configuration
upgrader := &websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
    EnableCompression: true,
}
```

## Migration Strategy

### Phase 1: Preparation
1. **Backup Current Implementation**
   - Create branch: `feature/websocket-migration`
   - Tag current state: `pre-gorilla-migration`

2. **Dependency Management**
   - Update go.mod to replace coder/websocket with gorilla/websocket
   - Run `go mod tidy` to clean up dependencies

### Phase 2: Core Implementation Changes

#### File: websocketconnection.go
**Priority: HIGH** - Core websocket functionality

**Changes Required:**
1. Update import statement
2. Modify `webSocketConnection` struct if needed
3. Implement context handling wrapper for gorilla websocket operations
4. Update `Write()` method to use `WriteMessage()`
5. Update `Read()` method to use `ReadMessage()`
6. Update `Close()` method signature if needed

**Implementation Notes:**
- Need to handle context cancellation manually since gorilla/websocket doesn't support context natively
- May need to create wrapper functions to maintain existing API compatibility

#### File: httpconnection.go
**Priority: HIGH** - Client connection establishment

**Changes Required:**
1. Update import statement
2. Replace `websocket.Dial()` with `websocket.Dialer.Dial()`
3. Convert `DialOptions` to `Dialer` configuration
4. Handle header setting differently

#### File: httpmux.go  
**Priority: HIGH** - Server connection handling

**Changes Required:**
1. Update import statement
2. Replace `websocket.Accept()` with `websocket.Upgrader.Upgrade()`
3. Convert `AcceptOptions` to `Upgrader` configuration
4. Update compression and origin checking logic

#### File: httpserver_test.go
**Priority: MEDIUM** - Test compatibility

**Changes Required:**
1. Update import statement
2. Update test websocket client code to use gorilla API
3. Ensure test scenarios cover all migration changes

#### File: serveroptions.go
**Priority: LOW** - Documentation updates

**Changes Required:**
1. Update documentation URLs from coder/websocket to gorilla/websocket
2. Update code comments to reflect new API

### Phase 3: Context Handling Implementation

**Challenge:** gorilla/websocket doesn't natively support context-based operations

**Solution Options:**
1. **Wrapper Approach**: Create wrapper functions that handle context cancellation
2. **Goroutine Approach**: Use goroutines with select statements for context handling
3. **Hybrid Approach**: Combine both strategies based on operation type

**Recommended Implementation:**
```go
// Example wrapper for context-aware read
func (w *webSocketConnection) readWithContext(ctx context.Context) (int, []byte, error) {
    type result struct {
        messageType int
        data        []byte
        err         error
    }
    
    resultChan := make(chan result, 1)
    go func() {
        messageType, data, err := w.conn.ReadMessage()
        resultChan <- result{messageType, data, err}
    }()
    
    select {
    case <-ctx.Done():
        w.conn.Close(websocket.CloseNormalClosure, "context cancelled")
        return 0, nil, ctx.Err()
    case r := <-resultChan:
        return r.messageType, r.data, r.err
    }
}
```

### Phase 4: Configuration Migration

#### WebSocket Message Types
**Mapping Required:**
```go
// coder/websocket -> gorilla/websocket
websocket.MessageText   -> websocket.TextMessage
websocket.MessageBinary -> websocket.BinaryMessage
```

#### Compression Settings
- coder: `CompressionContextTakeover` -> gorilla: `EnableCompression: true`

#### Origin Checking
- coder: `OriginPatterns` -> gorilla: `CheckOrigin` function

#### Security Settings
- coder: `InsecureSkipVerify` -> gorilla: Custom `CheckOrigin` implementation

### Phase 5: Testing Strategy

#### Unit Tests
1. **websocketconnection_test.go** - Test core connection functionality
2. **httpconnection_test.go** - Test client connection establishment  
3. **httpmux_test.go** - Test server upgrade handling

#### Integration Tests
1. **End-to-end connection tests** - Full SignalR protocol flow
2. **Transport negotiation tests** - Ensure websocket transport still works
3. **Error handling tests** - Context cancellation, connection failures

#### Performance Tests
1. **Benchmark comparisons** - Before/after migration performance
2. **Memory usage analysis** - Check for memory leaks
3. **Concurrent connection tests** - Stress testing

## Risk Assessment

### High Risk Areas
1. **Context Handling** - Manual implementation required for gorilla/websocket
2. **API Compatibility** - Existing connection interface must be maintained
3. **Error Handling** - Different error types and handling patterns

### Medium Risk Areas
1. **Configuration Options** - Option mapping between libraries
2. **Message Type Constants** - Different constant values
3. **Testing Coverage** - Ensuring all scenarios are tested

### Low Risk Areas
1. **Import Statements** - Straightforward replacement
2. **Documentation** - URL and comment updates

## Rollback Plan

### Immediate Rollback
- Revert to `pre-gorilla-migration` tag
- Restore original go.mod dependencies

### Gradual Rollback
- Maintain both implementations temporarily
- Feature flag to switch between websocket implementations
- Gradual migration of connections

## Success Criteria

### Functional Requirements
- [ ] All existing SignalR functionality works unchanged
- [ ] WebSocket transport negotiation successful
- [ ] Client and server connections establish correctly
- [ ] Message sending/receiving works for text and binary
- [ ] Connection cleanup and error handling preserved

### Performance Requirements
- [ ] No significant performance degradation
- [ ] Memory usage remains stable
- [ ] Connection latency within acceptable bounds

### Quality Requirements
- [ ] All tests pass
- [ ] No new linting errors or warnings
- [ ] Code coverage maintained or improved
- [ ] Documentation updated

## Timeline Estimate

### Development Phase (5-7 days)
- Day 1: Dependency update and basic import changes
- Day 2-3: Core websocketconnection.go implementation
- Day 4: httpconnection.go and httpmux.go updates
- Day 5: Context handling implementation
- Day 6-7: Testing and bug fixes

### Testing Phase (2-3 days)
- Unit testing and fixes
- Integration testing
- Performance validation

### Review and Documentation (1-2 days)
- Code review
- Documentation updates
- Final validation

**Total Estimated Time: 8-12 days**

## Next Steps

1. **Immediate Actions:**
   - Create migration branch
   - Update go.mod with gorilla/websocket dependency
   - Begin with websocketconnection.go implementation

2. **Team Coordination:**
   - Assign developers to specific files
   - Set up testing environment
   - Plan code review schedule

3. **Monitoring:**
   - Set up metrics for connection success rates
   - Monitor performance during migration
   - Track any user-reported issues
