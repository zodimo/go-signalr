# WebSocket Migration Execution Checklist

## Pre-Migration Setup

### ✅ Preparation Tasks
- [ ] **Create backup branch**
  ```bash
  git checkout -b feature/websocket-migration
  git tag pre-gorilla-migration
  ```

- [ ] **Document current state**
  - [ ] Note current test pass rate
  - [ ] Document any existing websocket issues
  - [ ] Record performance benchmarks

- [ ] **Set up testing environment**
  - [ ] Ensure all tests currently pass: `go test ./...`
  - [ ] Run chat sample to verify current functionality
  - [ ] Create test script for validation

## Phase 1: Dependency Management

### ✅ Update Dependencies
- [ ] **Remove old dependency**
  ```bash
  go mod edit -droprequire github.com/coder/websocket
  ```

- [ ] **Add new dependency**
  ```bash
  go get github.com/gorilla/websocket@latest
  ```

- [ ] **Clean up dependencies**
  ```bash
  go mod tidy
  ```

- [ ] **Verify no build errors after dependency change**
  ```bash
  go build ./...
  ```

## Phase 2: Create Helper Functions

### ✅ Context Support Implementation
- [ ] **Create websocket_context.go**
  - [ ] Add `contextualWebSocketConn` struct
  - [ ] Implement `WriteWithContext` method
  - [ ] Implement `ReadWithContext` method  
  - [ ] Implement `CloseWithReason` method
  - [ ] Add `wrapGorillaConn` function

- [ ] **Test helper functions**
  - [ ] Unit test for context cancellation
  - [ ] Unit test for timeout handling
  - [ ] Unit test for proper message passing

## Phase 3: Core File Updates

### ✅ websocketconnection.go (Priority: HIGH)
- [ ] **Update imports**
  - [ ] Change import to `github.com/gorilla/websocket`

- [ ] **Update types and structs**
  - [ ] Change `conn` field type to `*contextualWebSocketConn`
  - [ ] Update `NewWebSocketConnection` to use gorilla dialer

- [ ] **Update methods**
  - [ ] Fix `newWebSocketConnection` to wrap connection
  - [ ] Update `Write` method message type constants
  - [ ] Update `Write` method to use `WriteWithContext`
  - [ ] Update `Read` method to use `ReadWithContext`
  - [ ] Update close operations to use `CloseWithReason`

- [ ] **Test websocketconnection.go**
  ```bash
  go test -v -run WebSocket
  ```

### ✅ httpconnection.go (Priority: HIGH)
- [ ] **Update imports**
  - [ ] Change import to `github.com/gorilla/websocket`
  - [ ] Add `time` import if needed

- [ ] **Update websocket dialing logic (around line 153-181)**
  - [ ] Replace `websocket.DialOptions` with `websocket.Dialer`
  - [ ] Update `websocket.Dial` call to `dialer.DialContext`
  - [ ] Fix header handling for gorilla dialer

- [ ] **Test httpconnection.go**
  ```bash
  go test -v -run HTTPConnection
  ```

### ✅ httpmux.go (Priority: HIGH)
- [ ] **Update imports**
  - [ ] Change import to `github.com/gorilla/websocket`
  - [ ] Add `time` import if needed

- [ ] **Update websocket accept logic (around line 146-159)**
  - [ ] Replace `websocket.AcceptOptions` with `websocket.Upgrader`
  - [ ] Implement `CheckOrigin` function
  - [ ] Update compression settings
  - [ ] Replace `websocket.Accept` with `upgrader.Upgrade`
  - [ ] Update `SetReadLimit` call
  - [ ] Fix close operations to use `WriteControl` and `FormatCloseMessage`

- [ ] **Test httpmux.go**
  ```bash
  go test -v -run HTTPMux
  ```

### ✅ httpserver_test.go (Priority: MEDIUM)
- [ ] **Update imports**
  - [ ] Change import to `github.com/gorilla/websocket`

- [ ] **Update test websocket client code**
  - [ ] Replace websocket dial calls with gorilla dialer
  - [ ] Update message type constants in tests
  - [ ] Fix read/write operations in test code

- [ ] **Test httpserver_test.go**
  ```bash
  go test -v ./httpserver_test.go
  ```

### ✅ serveroptions.go (Priority: LOW)
- [ ] **Update documentation**
  - [ ] Change URLs from coder/websocket to gorilla/websocket (lines 63, 72)
  - [ ] Update comments to reflect new API

## Phase 4: Integration Testing

### ✅ Unit Tests
- [ ] **Run all tests**
  ```bash
  go test -v ./...
  ```

- [ ] **Run websocket-specific tests**
  ```bash
  go test -v -run WebSocket
  go test -v -run websocket
  ```

- [ ] **Fix any failing tests**
  - [ ] Document test failures
  - [ ] Implement fixes
  - [ ] Re-run tests to verify fixes

### ✅ Integration Tests
- [ ] **Test chat sample**
  ```bash
  cd chatsample && go run main.go
  ```
  - [ ] Open browser to test page
  - [ ] Test text message sending
  - [ ] Test multiple concurrent connections
  - [ ] Test connection cleanup on browser close

- [ ] **Test different transports**
  - [ ] WebSocket transport with text transfer
  - [ ] WebSocket transport with binary transfer
  - [ ] Server-Sent Events transport (should be unaffected)

### ✅ Performance Testing
- [ ] **Run benchmarks**
  ```bash
  go test -bench=. -benchmem
  ```

- [ ] **Compare with baseline**
  - [ ] Document any performance differences
  - [ ] Investigate significant performance regressions

## Phase 5: Error Handling & Edge Cases

### ✅ Error Scenarios
- [ ] **Test context cancellation**
  - [ ] Client connection timeout
  - [ ] Server connection timeout
  - [ ] Mid-operation cancellation

- [ ] **Test connection failures**
  - [ ] Invalid websocket URLs
  - [ ] Network disconnections
  - [ ] Server shutdown scenarios

- [ ] **Test origin checking**
  - [ ] Valid origins
  - [ ] Invalid origins
  - [ ] InsecureSkipVerify behavior

### ✅ Memory Leak Testing
- [ ] **Check for goroutine leaks**
  ```bash
  go test -v -run TestWebSocket -count=100
  ```

- [ ] **Check for connection leaks**
  - [ ] Monitor connection counts during stress testing
  - [ ] Verify proper cleanup on errors

## Phase 6: Documentation & Cleanup

### ✅ Code Quality
- [ ] **Run linters**
  ```bash
  golint ./...
  go vet ./...
  ```

- [ ] **Format code**
  ```bash
  go fmt ./...
  ```

- [ ] **Check for unused imports**
  ```bash
  goimports -w .
  ```

### ✅ Documentation Updates
- [ ] **Update README if needed**
  - [ ] Note websocket library change
  - [ ] Update any examples that reference websocket usage

- [ ] **Update CHANGELOG**
  - [ ] Document breaking changes
  - [ ] Note the library migration

- [ ] **Code comments**
  - [ ] Remove any TODO comments added during migration
  - [ ] Update function documentation if APIs changed

## Phase 7: Final Validation

### ✅ Comprehensive Testing
- [ ] **Run full test suite multiple times**
  ```bash
  for i in {1..5}; do go test ./... && echo "Run $i: PASS" || echo "Run $i: FAIL"; done
  ```

- [ ] **Manual testing checklist**
  - [ ] Start chat sample server
  - [ ] Connect 5+ concurrent clients
  - [ ] Send 100+ messages rapidly
  - [ ] Close connections gracefully and abruptly
  - [ ] Test on different browsers
  - [ ] Test with different network conditions

- [ ] **Load testing**
  - [ ] Test with high connection count
  - [ ] Test with high message volume
  - [ ] Monitor resource usage

### ✅ Rollback Verification
- [ ] **Test rollback procedure**
  ```bash
  git checkout pre-gorilla-migration
  go test ./...
  # Verify everything still works
  git checkout feature/websocket-migration
  ```

- [ ] **Document rollback steps**
  - [ ] Clear instructions for emergency rollback
  - [ ] Identify monitoring metrics to watch

## Phase 8: Deployment Preparation

### ✅ Pre-Deployment
- [ ] **Create deployment checklist**
  - [ ] Dependencies to update in production
  - [ ] Configuration changes needed
  - [ ] Monitoring metrics to watch

- [ ] **Create monitoring alerts**
  - [ ] WebSocket connection failures
  - [ ] WebSocket connection count
  - [ ] WebSocket message processing time

- [ ] **Performance baseline**
  - [ ] Document expected performance characteristics
  - [ ] Create performance regression tests

### ✅ Post-Deployment Monitoring
- [ ] **Immediate checks (first hour)**
  - [ ] WebSocket connections establishing successfully
  - [ ] No increase in error rates
  - [ ] Performance within expected ranges

- [ ] **Short-term monitoring (first day)**
  - [ ] Memory usage stable
  - [ ] No connection leaks
  - [ ] Client functionality working correctly

- [ ] **Long-term monitoring (first week)**
  - [ ] No degradation in user experience
  - [ ] All transport types working correctly
  - [ ] Performance metrics stable

## Emergency Procedures

### 🚨 If Issues Occur
- [ ] **Immediate rollback plan**
  ```bash
  git revert [migration-commit-hash]
  # Or full rollback:
  git checkout pre-gorilla-migration
  ```

- [ ] **Issue triage checklist**
  - [ ] Identify affected functionality
  - [ ] Check logs for gorilla/websocket specific errors
  - [ ] Compare behavior with coder/websocket version
  - [ ] Document issue for post-mortem

### 📊 Success Metrics
- [ ] **All tests passing**: ✅/❌
- [ ] **Chat sample working**: ✅/❌
- [ ] **Performance maintained**: ✅/❌
- [ ] **No memory leaks**: ✅/❌
- [ ] **All transports working**: ✅/❌

## Final Sign-off

### ✅ Migration Complete
- [ ] **Technical lead approval**
  - [ ] Code review completed
  - [ ] All tests passing
  - [ ] Performance acceptable

- [ ] **QA approval**
  - [ ] Manual testing completed
  - [ ] Edge cases tested
  - [ ] Documentation updated

- [ ] **Stakeholder approval**
  - [ ] Migration goals achieved
  - [ ] No regression in functionality
  - [ ] Ready for production deployment

**Migration Completed on**: _______________  
**Completed by**: _______________  
**Reviewed by**: _______________
