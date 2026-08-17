# E2E Testing Guide for gRPC File Sync Changes

## Overview

This guide outlines how to perform comprehensive end-to-end testing to validate that the gRPC file sync implementation works correctly in production scenarios.

## Test Levels

### 1. Unit/Integration Test (Already Passing ✅)
**Test**: `CLS-003` - File Sync gRPC Streaming Performance
**Purpose**: Validates gRPC streaming implementation directly
**Command**: 
```bash
cd tests
./cli.py --spec CLS-003
```
**What it tests**:
- gRPC server availability and connectivity
- Concurrent request handling (500 simultaneous requests)
- File scanning with 1000 files
- Performance comparison (HTTP vs gRPC)
- Error handling and reliability

**Status**: ✅ Passing with 100% success rate under extreme load

---

### 2. Task Execution E2E Tests (Recommended)

These tests validate the complete workflow: spider creation → task execution → file sync → task completion.

#### Option A: UI-Based E2E Test (Comprehensive)
**Test**: `UI-001` + `UI-003` combination
**Purpose**: Validates complete user workflow through web interface

**Command**:
```bash
cd tests
# Test spider management
./cli.py --spec UI-001

# Test task execution with file sync
./cli.py --spec UI-003
```

**What it tests**:
1. **Spider Creation** (UI-001):
   - Create spider with files
   - Upload/edit spider files
   - File synchronization to workers

2. **Task Execution** (UI-003):
   - Run task on spider
   - File sync from master to worker (uses gRPC)
   - Task execution with synced files
   - View task logs
   - Verify task completion

**File Sync Points**:
- When task starts, worker requests files from master via gRPC
- gRPC streaming sends file metadata and content
- Worker receives and writes files to local workspace
- Task executes with synced files

**How to verify gRPC is working**:
```bash
# During test execution, check master logs for gRPC activity
docker logs astiango_hub_master 2>&1 | grep -i "grpc\|sync\|stream"

# Should see messages like:
# "performing directory scan for /root/astiango_hub_workspace/spider-id"
# "scanned N files from path"
# "streaming files to worker"
```

#### Option B: Cluster Node Reconnection Test
**Test**: `CLS-001` - Master-Worker Node Disconnection
**Purpose**: Validates file sync during node reconnection scenarios

**Command**:
```bash
cd tests
./cli.py --spec CLS-001
```

**What it tests**:
- Worker disconnection and reconnection
- File resync after worker comes back online
- Task execution after reconnection (requires file sync)

---

### 3. Manual E2E Validation Steps

For thorough validation, perform these manual steps:

#### Step 1: Create Test Spider
```bash
# Via UI or API
curl -X POST http://localhost:8080/api/spiders \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-grpc-sync",
    "cmd": "python main.py",
    "project_id": "default"
  }'
```

#### Step 2: Add Files to Spider
- Add multiple files (at least 10-20)
- Include various file types (.py, .txt, .json)
- Mix of small and larger files

#### Step 3: Run Task and Monitor
```bash
# Start task
curl -X POST http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"spider_id": "test-grpc-sync"}'

# Watch master logs for gRPC activity
docker logs -f astiango_hub_master | grep -i "sync\|grpc"

# Expected output:
# [SyncServiceServer] performing directory scan for /root/astiango_hub_workspace/test-grpc-sync
# [SyncServiceServer] scanned 20 files from /root/astiango_hub_workspace/test-grpc-sync
# (multiple concurrent requests may show deduplication)
```

#### Step 4: Verify Worker Received Files
```bash
# Check worker container
docker exec astiango_hub_worker ls -la /root/astiango_hub_workspace/test-grpc-sync/

# Should see all spider files present
```

#### Step 5: Check Task Execution
- Task should complete successfully
- Task logs should show file access working
- No "file not found" errors

---

### 4. High-Concurrency Stress Test

To validate production readiness under load:

#### Step 1: Prepare Multiple Spiders
- Create 5-10 different spiders
- Each with 50-100 files

#### Step 2: Trigger Concurrent Tasks
```bash
# Run multiple tasks simultaneously
for i in {1..20}; do
  curl -X POST http://localhost:8080/api/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"spider_id\": \"spider-$((i % 5))\"}" &
done
wait

# Or use the test framework
cd tests
./cli.py --spec CLS-003  # Already tests 500 concurrent
```

#### Step 3: Monitor System Behavior
```bash
# Check gRPC deduplication is working
docker logs astiango_hub_master | grep "notified.*subscribers"

# Should see messages like:
# "scan complete, notified 5 subscribers"
# (Proves deduplication: 1 scan served multiple requests)

# Monitor resource usage
docker stats astiango_hub_master astiango_hub_worker
```

**Success Criteria**:
- All tasks complete successfully (100% success rate)
- No "file not found" errors
- Master CPU/memory remains stable
- Evidence of request deduplication in logs

---

### 5. Regression Testing

Run the complete test suite to ensure no regressions:

```bash
cd tests

# Run all cluster tests
./cli.py --spec CLS-001  # Node disconnection
./cli.py --spec CLS-002  # Docker container recovery  
./cli.py --spec CLS-003  # gRPC performance (stress test)

# Run scheduler tests (tasks depend on file sync)
./cli.py --spec SCH-001  # Task status reconciliation

# Run UI tests (complete workflows)
./cli.py --spec UI-001   # Spider management
./cli.py --spec UI-003   # Task management
```

---

## Environment Variables for Testing

To explicitly test gRPC vs HTTP modes:

### Test with gRPC Enabled (Default)
```bash
# Master
ASTIANGO_GRPC_ENABLED=true
ASTIANGO_GRPC_ADDRESS=:9666

# Worker
ASTIANGO_GRPC_ENABLED=true
ASTIANGO_GRPC_ADDRESS=master:9666
```

### Test with HTTP Fallback (Validation)
```bash
# Master
ASTIANGO_GRPC_ENABLED=false

# Worker
ASTIANGO_GRPC_ENABLED=false
```

Run the same tests in both modes to verify:
1. gRPC mode: High performance, 100% reliability
2. HTTP mode: Works but lower performance under load

---

## Expected Results

### gRPC Mode (Production)
- ✅ 100% task success rate under load
- ✅ Fast file sync (< 1s for 1000 files, 500 concurrent)
- ✅ Request deduplication working (logs show "notified N subscribers")
- ✅ Low master CPU/memory usage
- ✅ No JSON parsing errors
- ✅ Streaming file transfer

### HTTP Mode (Legacy/Fallback)
- ⚠️ Lower success rate under high load (37% at 500 concurrent)
- ⚠️ Slower sync (20-30s for 1000 files, 500 concurrent)
- ⚠️ No deduplication (each request = separate scan)
- ⚠️ Higher master resource usage
- ⚠️ Potential JSON parsing errors at high concurrency

---

## Troubleshooting E2E Tests

### Issue: Tasks Fail with "File Not Found"
**Diagnosis**:
```bash
# Check if gRPC server is running
docker exec astiango_hub_master netstat -tlnp | grep 9666

# Check worker can reach master
docker exec astiango_hub_worker nc -zv master 9666

# Check logs for sync errors
docker logs astiango_hub_master | grep -i "sync.*error"
```

**Solutions**:
- Verify `ASTIANGO_GRPC_ENABLED=true` on both master and worker
- Check network connectivity between containers
- Verify workspace paths match (`/root/astiango_hub_workspace`)

### Issue: gRPC Requests Fail
**Diagnosis**:
```bash
# Check gRPC server logs
docker logs astiango_hub_master | grep "SyncServiceServer"

# Test gRPC connectivity from host
grpcurl -plaintext localhost:9666 list
```

**Solutions**:
- Verify protobuf versions match (Python 6.x, Go latest)
- Check authentication key matches between master and worker
- Verify port 9666 is exposed and mapped correctly

### Issue: Poor Performance Despite gRPC
**Diagnosis**:
```bash
# Check if deduplication is working
docker logs astiango_hub_master | grep "notified.*subscribers"

# If no deduplication messages, check cache settings
```

**Solutions**:
- Verify gRPC cache TTL is set (60s default)
- Check concurrent requests are actually happening simultaneously
- Monitor for network bandwidth limits

---

## CI/CD Integration

Add these tests to your CI pipeline:

```yaml
# .github/workflows/test.yml
jobs:
  e2e-grpc-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Setup AstianGO Hub
        run: docker-compose up -d
        
      - name: Wait for services
        run: sleep 30
        
      - name: Run gRPC performance test
        run: cd tests && ./cli.py --spec CLS-003
        
      - name: Run cluster tests
        run: |
          cd tests
          ./cli.py --spec CLS-001
          ./cli.py --spec CLS-002
          
      - name: Run UI E2E tests
        run: |
          cd tests  
          ./cli.py --spec UI-001
          ./cli.py --spec UI-003
```

---

## Conclusion

**Recommended Test Sequence**:
1. ✅ `CLS-003` - Validates gRPC implementation directly (already passing)
2. 🔄 `UI-001` + `UI-003` - Validates complete user workflow with file sync
3. 🔄 `CLS-001` + `CLS-002` - Validates file sync in cluster scenarios
4. 🔄 Manual validation - Create spider, run tasks, verify files synced
5. 🔄 Stress test - Run many concurrent tasks, verify 100% success

All tests should show **100% success rate** with gRPC enabled, validating production readiness.
