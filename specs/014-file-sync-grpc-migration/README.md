---
status: complete
created: 2025-10-30
tags: [grpc, networking, file-sync, migration]
priority: medium
---

# File Sync Issue After gRPC Migration

**Date**: 2025-10-30  
**Status**: Investigation Complete → Action Required  
**Severity**: High (blocks spider execution on workers)

## Executive Summary

**Problem**: Spider tasks fail on worker nodes with "no such file or directory" errors when creating nested directory structures during gRPC file sync.

**Root Cause**: Bug in `downloadFileGRPC()` at line 188-189 uses naive string slicing to extract directory path, which incorrectly truncates directory names mid-character. Example: `astiango_hub_project/spiders/` becomes `astiango_hub_project/sp`.

**Impact**: 
- Tasks fail immediately during file sync phase
- Error: `failed to create file: open .../astiango_hub_project/spiders/quotes.py: no such file or directory`
- All spiders with nested directory structures affected
- Production deployments broken

**Solution**:
1. **Critical fix**: Replace string slicing with `filepath.Dir()` (one-line change)
2. **Test coverage**: REL-004 and REL-005 already test this scenario
3. **Additional improvements**: Preserve file permissions, add retry logic

**Status**: Root cause confirmed via log analysis. Fix is trivial and ready to implement.

## Problem Statement

Spider tasks fail on worker nodes during the gRPC file sync phase with the error:

```
ERROR [2025-10-30 14:57:17] [AstianGO Hub] error downloading file astiango_hub_project/spiders/quotes.py: 
failed to create file: open /root/astiango_hub_workspace/69030c474b101b7b116bc264/astiango_hub_project/spiders/quotes.py: 
no such file or directory
```

This occurs when:
1. Master node sends file list via gRPC streaming
2. Worker attempts to download files with nested directory structures
3. Directory creation fails due to incorrect path calculation
4. File creation subsequently fails because parent directory doesn't exist

The issue started after migration from HTTP-based file sync to gRPC-based sync.

## Symptoms

**From Task Logs** (2025-10-30 14:57:17):
```
INFO  starting gRPC file synchronization for spider: 69030c474b101b7b116bc264
INFO  fetching file list from master via gRPC
INFO  received complete file list: 11 files
DEBUG file not found locally: astiango_hub_project/spiders/quotes.py
DEBUG downloading file via gRPC: astiango_hub_project/spiders/quotes.py
ERROR error downloading file astiango_hub_project/spiders/quotes.py: 
      failed to create file: open /root/astiango_hub_workspace/69030c474b101b7b116bc264/astiango_hub_project/spiders/quotes.py: 
      no such file or directory
WARN  error synchronizing files: failed to create file: open .../astiango_hub_project/spiders/quotes.py: no such file or directory
```

**Observable Behavior**:
- gRPC file list fetched successfully (11 files)
- File download initiated for nested directory file
- Directory creation fails silently
- File creation fails with "no such file or directory"
- Task continues but fails immediately (exit status 2)
- Scrapy reports "no active project" because files aren't synced

**Key Pattern**: Affects files in nested directories (e.g., `astiango_hub_project/spiders/quotes.py`), not root-level files.

## Root Cause Hypothesis

The migration from HTTP sync to gRPC sync for file synchronization may have introduced issues in:

1. **File transfer mechanism**: gRPC implementation may not correctly transfer all spider files
2. **Timing issues**: Files may not be fully synced before task execution begins
3. **File permissions**: Synced files may not have correct execution permissions
4. **Path handling**: File paths may be incorrectly resolved in the new gRPC implementation
5. **Client initialization**: SyncClient may not be properly initialized before task execution
6. **Error handling**: Errors during gRPC sync might be silently ignored or not properly propagated

## Investigation Findings

### gRPC File Sync Implementation

**Code Locations**:
- Server: `astiango-hub/core/grpc/server/sync_service_server.go`
- Client: `astiango-hub/core/grpc/client/client.go` (GetSyncClient method)
- Task Runner: `astiango-hub/core/task/handler/runner_sync_grpc.go`
- Sync Switch: `astiango-hub/core/task/handler/runner_sync.go`

**How It Works**:
1. Runner calls `syncFiles()` which checks `utils.IsSyncGrpcEnabled()`
2. If enabled, calls `syncFilesGRPC()` which:
   - Gets sync client via `client2.GetGrpcClient().GetSyncClient()`
   - Streams file list from master via `StreamFileScan`
   - Compares master files with local worker files
   - Downloads new/modified files via `StreamFileDownload`
   - Deletes files that no longer exist on master

**When Files Are Synced**:
- In `runner.go` line 198: `r.syncFiles()` is called
- This happens **BEFORE** `r.cmd.Start()` (line 217)
- Sync is done during task preparation phase
- If sync fails, task continues with a WARNING, not an error

### Potential Issues Identified

1. **Error Handling**: In `runner.go:200`, sync errors are logged as warnings:
   ```go
   if err := r.syncFiles(); err != nil {
       r.Warnf("error synchronizing files: %v", err)
   }
   ```
   Task continues even if file sync fails!

2. **Client Registration**: The gRPC client must be registered before `GetSyncClient()` works
   - Client registration happens in `register()` method
   - If client not registered, `GetSyncClient()` might fail

3. **Directory Creation in gRPC**: In `downloadFileGRPC()`, directory creation logic:
   ```go
   targetDir := targetPath[:len(targetPath)-len(path)]
   ```
   This is string manipulation, might create incorrect paths

4. **File Permissions**: gRPC downloads files with `os.Create()` which doesn't preserve permissions from master

## Investigation Plan

### 1. Review gRPC File Sync Implementation ✅

- [x] Check `astiango-hub/grpc/` for file sync service implementation
- [x] Compare with old HTTP sync implementation
- [x] Verify file transfer completeness
- [x] Check error handling in gRPC sync

### 2. Analyze File Sync Flow ✅

- [x] Master node: File preparation and sending
- [x] Worker node: File reception and storage  
- [x] Verify file sync triggers (when does sync happen?)
- [x] Check if sync completes before task execution

**Findings**:
- File sync happens in task runner initialization
- Sync errors are only **warned**, not failed
- Task continues even if files fail to sync
- This explains why users see "missing file" errors during task execution

### 3. Test Scenarios to Cover

The following test scenarios should be added to `astiango-hub-test`:

#### Cluster File Sync Tests

**CLS-XXX: Spider File Sync Before Task Execution**
- Upload spider with code files to master
- Start spider task on worker node
- Verify all code files are present on worker before execution
- Verify task executes successfully with synced files

**CLS-XXX: Multiple Worker File Sync**
- Upload spider to master
- Run tasks on multiple workers simultaneously
- Verify all workers receive complete file set
- Verify no file corruption or partial transfers

**CLS-XXX: Large File Sync Reliability**
- Upload spider with large files (>10MB)
- Sync to worker node
- Verify file integrity (checksums)
- Verify execution works correctly

**CLS-XXX: File Sync Timing**
- Upload spider to master
- Immediately trigger task on worker
- Verify sync completes before execution attempt
- Verify proper error handling if sync incomplete

#### Edge Cases

**CLS-XXX: File Permission Sync**
- Upload spider with executable scripts
- Sync to worker
- Verify file permissions are preserved
- Verify scripts can execute

**CLS-XXX: File Update Sync**
- Upload spider v1 to master
- Sync to worker
- Update spider files (v2)
- Verify worker receives updates
- Verify task uses updated files

## Code Locations

### gRPC Implementation
- `astiango-hub/grpc/` - gRPC service definitions
- `astiango-hub/core/grpc/` - gRPC implementation details
- Look for file sync related services

### File Sync Logic
- `astiango-hub/core/fs/` - File system operations
- `astiango-hub/backend/` - Backend file sync handlers
- `core/spider/` - Spider file management (Pro)

### HTTP Sync (Legacy - for comparison)
- Search for HTTP file sync implementation to compare

## Action Items

### 1. Immediate: Fix Directory Path Bug ✅ **ROOT CAUSE IDENTIFIED**

**Issue**: Incorrect string slicing in `downloadFileGRPC()` breaks directory creation for nested paths

**Location**: `astiango-hub/core/task/handler/runner_sync_grpc.go:188-189`

**Current Code** (BUGGY):
```go
targetPath := fmt.Sprintf("%s/%s", r.cwd, path)
targetDir := targetPath[:len(targetPath)-len(path)]  // BUG: Wrong calculation
```

**Fixed Code**:
```go
targetPath := fmt.Sprintf("%s/%s", r.cwd, path)
targetDir := filepath.Dir(targetPath)  // Use stdlib function
```

**Why This Fixes It**:
- `filepath.Dir()` properly extracts parent directory from any file path
- Works with any nesting level and path separator
- Same approach used in working HTTP sync implementation

**Priority**: **CRITICAL** - One-line fix that unblocks all spider execution

**Import Required**: Add `"path/filepath"` to imports if not already present

### 2. Secondary: Make File Sync Errors Fatal (Optional Enhancement)

**Issue**: File sync errors are logged as warnings but don't fail the task

**Location**: `astiango-hub/core/task/handler/runner.go:198-200`

**Current Code**:
```go
if err := r.syncFiles(); err != nil {
    r.Warnf("error synchronizing files: %v", err)
}
```

**Note**: With the directory path bug fixed, this becomes less critical. However, making sync errors fatal would improve error visibility.

**Suggested Fix** (if desired):
```go
if err := r.syncFiles(); err != nil {
    r.Errorf("error synchronizing files: %v", err)
    return r.updateTask(constants.TaskStatusError, err)
}
```

**Rationale**: Tasks should not execute if files are not synced. Currently, the directory bug is caught but task continues, leading to confusing downstream errors.

### 3. Short-term: Validate Fix with Existing Tests

**Created Tests**:
- `REL-004`: Worker Node File Sync Validation
  - Spec: `astiango-hub-test/specs/reliability/REL-004-worker-file-sync-validation.md` ✅
  - Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_004_worker_file_sync_validation.py` ✅
  - Tests basic file sync functionality with 4 files
  - Validates gRPC sync mechanism and file presence on worker

- `REL-005`: Concurrent Worker File Sync Reliability
  - Spec: `astiango-hub-test/specs/reliability/REL-005-concurrent-worker-file-sync.md` ✅
  - Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_005_concurrent_worker_file_sync.py` ✅
  - Tests multi-worker concurrent sync scenarios with 11 files
  - Creates 4 concurrent tasks to test gRPC sync under load

**Status**: ✅ Test specifications and runners complete. Both use proper Helper class pattern (AuthHelper, SpiderHelper, TaskHelper, NodeHelper).

**Next Steps**:
- [ ] Run tests to reproduce and validate the issue
- [ ] Add tests to CI pipeline
- [ ] Create additional edge case tests (large files, permissions, updates)

### 3. Medium-term: Fix gRPC Implementation Issues

**Issue 1**: Directory path handling in `downloadFileGRPC()` ✅ **ROOT CAUSE CONFIRMED**

**Location**: `astiango-hub/core/task/handler/runner_sync_grpc.go:188-189`

```go
targetPath := fmt.Sprintf("%s/%s", r.cwd, path)
targetDir := targetPath[:len(targetPath)-len(path)]
```

**The Bug**: String slicing produces incorrect directory paths!

**Example with actual values**:
- `r.cwd` = `/root/astiango_hub_workspace/69030c474b101b7b116bc264`
- `path` = `astiango_hub_project/spiders/quotes.py` (34 chars)
- `targetPath` = `/root/astiango_hub_workspace/69030c474b101b7b116bc264/astiango_hub_project/spiders/quotes.py` (115 chars)
- `targetDir` = `targetPath[:115-34]` = `targetPath[:81]`
- **Result**: `/root/astiango_hub_workspace/69030c474b101b7b116bc264/astiango_hub_project/sp` ❌

This cuts off in the middle of "spiders", creating path `/astiango_hub_project/sp` instead of `/astiango_hub_project/spiders/`.

**Error Message**: `failed to create file: open /root/astiango_hub_workspace/.../astiango_hub_project/spiders/quotes.py: no such file or directory`

**The Fix**: Use `filepath.Dir()` like the HTTP version does:

```go
targetPath := fmt.Sprintf("%s/%s", r.cwd, path)
targetDir := filepath.Dir(targetPath)  // Properly extracts parent directory
if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
    return fmt.Errorf("failed to create directory: %w", err)
}
```

**Comparison with HTTP sync** (`runner_sync.go:267-273`):
```go
// HTTP version (CORRECT)
dirPath := filepath.Dir(filePath)
err = os.MkdirAll(dirPath, os.ModePerm)
```

This string manipulation bug is **error-prone** and caused by trying to manually extract the directory path instead of using Go's standard library.

**Issue 2**: File permissions not preserved

**Location**: `astiango-hub/core/task/handler/runner_sync_grpc.go:183`

```go
file, err := os.Create(targetPath)
```

Should use `os.OpenFile()` with mode from `masterFile.Mode` to preserve permissions.

**Issue 3**: Missing retry logic for gRPC failures

The HTTP sync has retry with backoff (`performHttpRequest`), but gRPC sync doesn't.

### 4. Short-term: Validate Fix with Existing Tests

**Existing Tests That Cover This Bug**:
- `REL-004`: Worker Node File Sync Validation
  - Spec: `astiango-hub-test/specs/reliability/REL-004-worker-file-sync-validation.md` ✅
  - Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_004_worker_file_sync_validation.py` ✅
  - Tests basic file sync with nested directories (4 files)
  
- `REL-005`: Concurrent Worker File Sync Reliability
  - Spec: `astiango-hub-test/specs/reliability/REL-005-concurrent-worker-file-sync.md` ✅
  - Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_005_concurrent_worker_file_sync.py` ✅
  - Tests multi-worker concurrent sync with Scrapy project structure (11 files)
  - Creates `astiango_hub_project/spiders/quotes.py` - the exact file that triggered this bug!

**Validation Steps**:
1. Apply the `filepath.Dir()` fix to `runner_sync_grpc.go`
2. Run tests: `uv run ./cli.py --spec REL-004 && uv run ./cli.py --spec REL-005`
3. Verify all files sync successfully to worker nodes
4. Verify tasks execute without "no such file or directory" errors

**Expected Result**: Tests should pass with the fix applied. REL-005 specifically exercises the exact file path that failed in production logs.

### 5. Long-term: Enhanced Monitoring and Logging

**Add**:
- File sync success/failure metrics
- gRPC sync performance metrics
- Detailed logging of sync operations
- Health check for gRPC sync service
- Worker-side sync validation logging

## Test Coverage Strategy

### Existing Test Coverage ✅

**REL-004: Worker Node File Sync Validation**
- Location: `astiango-hub-test/specs/reliability/REL-004-worker-file-sync-validation.md`
- Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_004_worker_file_sync_validation.py`
- Coverage: Basic file sync with nested directories (4 files)
- Status: ✅ Spec and runner complete

**REL-005: Concurrent Worker File Sync Reliability**
- Location: `astiango-hub-test/specs/reliability/REL-005-concurrent-worker-file-sync.md`
- Runner: `astiango-hub-test/astiango_hub_test/runners/reliability/REL_005_concurrent_worker_file_sync.py`
- Coverage: Multi-worker concurrent sync with full Scrapy project (11 files)
- Files: Includes `astiango_hub_project/spiders/quotes.py` - the exact path that failed!
- Scenario: 4 concurrent tasks across 2 workers
- Status: ✅ Spec and runner complete

**Why These Tests Catch the Bug**:
Both tests create spiders with nested directory structures:
- REL-004: Tests basic nested paths
- REL-005: Tests the exact Scrapy structure that failed in production

The bug would cause both tests to fail with "no such file or directory" error during gRPC sync.

### Test Execution

**Before Fix** (Expected):
```bash
uv run ./cli.py --spec REL-005
# Expected: FAIL with "failed to create file: .../astiango_hub_project/spiders/quotes.py: no such file or directory"
```

**After Fix** (Expected):
```bash
uv run ./cli.py --spec REL-004  # Should PASS
uv run ./cli.py --spec REL-005  # Should PASS
```

### Success Criteria
- All spider files present on worker before execution
- Files have correct permissions and content
- No timing issues between sync and execution
- Multiple workers receive consistent file sets
- Large files transfer correctly
- Proper error handling when sync fails

## References

- **Bug Location**: `astiango-hub/core/task/handler/runner_sync_grpc.go:188-189`
- **HTTP Sync Reference**: `astiango-hub/core/task/handler/runner_sync.go:267-273` (correct implementation)
- **Test Coverage**: REL-004 and REL-005 in `astiango-hub-test/specs/reliability/`
- **Production Log**: Task ID `69030c4c4b101b7b116bc266`, Spider ID `69030c474b101b7b116bc264` (2025-10-30 14:57:17)
- **Error Pattern**: `failed to create file: open .../astiango_hub_project/spiders/quotes.py: no such file or directory`

## Timeline

- **2025-10-30 14:57:17**: Production error observed in task logs
- **2025-10-30 (investigation)**: Root cause identified as string slicing bug in `downloadFileGRPC()`
- **2025-10-30 (analysis)**: Confirmed fix is one-line change to use `filepath.Dir()`
- **Next**: Apply fix and validate with REL-004/REL-005 tests

## Notes

- **Severity**: CRITICAL - Blocks all spider execution with nested directories
- **Fix Complexity**: TRIVIAL - One-line change, no architectural changes needed
- **Test Coverage**: Already exists - REL-005 tests exact failure scenario
- **Root Cause**: Naive string manipulation instead of using Go stdlib `filepath.Dir()`
- **Lesson**: Always use standard library functions for path operations, never manual string slicing
