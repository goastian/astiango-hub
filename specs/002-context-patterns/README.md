---
status: complete
created: 2025-09-30
tags: [architecture]
priority: medium
---

# Context Usage Patterns - Best Practices Guide

## 🎯 When to Use `context.Background()`

### ✅ Acceptable Usage
- **Application initialization**: Setting up services, database connections at startup
- **Test files**: Unit tests and integration tests  
- **Background job scheduling**: One-time setup of cron jobs or schedulers
- **OAuth/Authentication flows**: Creating authenticated clients (not the requests themselves)

### 🚫 Avoid `context.Background()` in:
- **HTTP handlers**: Use `c.Request.Context()` (Gin) or `r.Context()` (net/http)
- **gRPC method implementations**: Use the provided context parameter
- **Long-running operations**: Database queries, external API calls, file I/O
- **Operations called from other contextual operations**

## 🏗️ Recommended Patterns

### HTTP Request Handlers
```go
func PostLLMChatStream(c *gin.Context, params *PostLLMChatParams) error {
    // ✅ Use request context with timeout
    ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
    defer cancel()
    
    return llmProvider.GenerateStream(ctx, messages)
}
```

### Background Tasks with Lifecycle
```go
type TaskHandler struct {
    ctx    context.Context
    cancel context.CancelFunc
}

func (h *TaskHandler) Start() {
    // ✅ Create cancellable context for service lifecycle
    h.ctx, h.cancel = context.WithCancel(context.Background())
    
    go func() {
        for {
            // ✅ Use service context with timeout for individual operations
            taskCtx, taskCancel := context.WithTimeout(h.ctx, 10*time.Minute)
            err := h.processTask(taskCtx)
            taskCancel()
            
            select {
            case <-h.ctx.Done():
                return // Service shutting down
            default:
                // Continue processing
            }
        }
    }()
}

func (h *TaskHandler) Stop() {
    h.cancel() // Cancel all operations
}
```

### Database Operations
```go
// ✅ Accept context parameter
func (s *Service) FindUser(ctx context.Context, id primitive.ObjectID) (*User, error) {
    // Database operations automatically inherit timeout/cancellation
    return s.userCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
}

// ✅ Wrapper with default timeout (use sparingly)
func (s *Service) FindUserWithDefaultTimeout(id primitive.ObjectID) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return s.FindUser(ctx, id)
}
```

### gRPC Client Calls
```go
func (c *Client) Connect(ctx context.Context) error {
    // ✅ Use provided context
    conn, err := grpc.DialContext(ctx, c.address, c.dialOptions...)
    if err != nil {
        return err
    }
    c.conn = conn
    return nil
}

// ✅ Wrapper with timeout for convenience
func (c *Client) ConnectWithTimeout(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    return c.Connect(ctx)
}
```

### Stream Operations
```go
func (s *Service) SubscribeToEvents(ctx context.Context) error {
    stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{})
    if err != nil {
        return err
    }
    
    go func() {
        defer stream.CloseSend()
        for {
            msg, err := stream.Recv()
            if err != nil {
                return
            }
            
            select {
            case s.eventChan <- msg:
            case <-ctx.Done():
                return // Context cancelled
            }
        }
    }()
    
    return nil
}
```

## 🚨 Common Anti-Patterns

### ❌ Breaking Context Chain
```go
// DON'T do this - breaks cancellation
func (h *Handler) ProcessRequest(ctx context.Context) error {
    // This loses the original context!
    return h.doWork(context.Background()) 
}

// ✅ DO this instead
func (h *Handler) ProcessRequest(ctx context.Context) error {
    return h.doWork(ctx)
}
```

### ❌ No Timeout for External Calls
```go
// DON'T do this - can hang forever
func CallExternalAPI() error {
    return httpClient.Get(context.Background(), url)
}

// ✅ DO this instead  
func CallExternalAPI() error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return httpClient.Get(ctx, url)
}
```

### ❌ Not Handling Context Cancellation
```go
// DON'T do this - goroutine can leak
func ProcessItems(ctx context.Context, items []Item) {
    for _, item := range items {
        processItem(item) // Can't be cancelled
    }
}

// ✅ DO this instead
func ProcessItems(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := processItem(ctx, item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### ❌ Using Parent Context for Async Operations
```go
// DON'T do this - async operation tied to parent lifecycle
func (r *Runner) finish() {
    // ... main work completed ...
    
    // This notification may fail if r.ctx gets cancelled
    go func() {
        ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second) // ❌ Problem!
        defer cancel()
        sendNotification(ctx)
    }()
}

// ✅ DO this instead - independent context for async operations
func (r *Runner) finish() {
    // ... main work completed ...
    
    // Use independent context for fire-and-forget operations
    go func() {
        // Preserve important values if needed
        taskID := r.taskID
        correlationID := r.ctx.Value("correlationID")
        
        // Create independent context with its own timeout
        ctx := context.WithValue(context.Background(), "correlationID", correlationID)
        ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
        defer cancel()
        
        sendNotification(ctx, taskID)
    }()
}
```

## 🔄 Async Operations Context Strategy

### When to Use Independent Context

Async operations need **independent context** when they have different lifecycles than their parent:

#### ✅ Use Independent Context For:
- **Background notifications**: Email, webhooks, push notifications
- **Post-processing tasks**: Log persistence, metrics collection, cleanup
- **Fire-and-forget operations**: Operations that should complete regardless of parent status
- **Operations with different timeouts**: When async operation needs longer/shorter timeout than parent

#### 🔗 Use Parent Context For:
- **Synchronous operations**: Part of the main request/response cycle  
- **Real-time operations**: Must be cancelled when parent is cancelled
- **Resource-bound operations**: Should stop immediately when parent stops

### Async Context Patterns

#### Pattern 1: Complete Independence
```go
func (s *Service) handleRequest(ctx context.Context) error {
    // Process main request
    result := s.processRequest(ctx)
    
    // Async operation with independent lifecycle
    go func() {
        // Create completely independent context
        asyncCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
        defer cancel()
        
        s.logToDatabase(asyncCtx, result)
    }()
    
    return nil
}
```

#### Pattern 2: Value Preservation
```go
func (s *Service) handleRequest(ctx context.Context) error {
    result := s.processRequest(ctx)
    
    go func() {
        // Preserve important values while creating independent context
        userID := ctx.Value("userID")
        traceID := ctx.Value("traceID")
        
        asyncCtx := context.WithValue(context.Background(), "userID", userID)
        asyncCtx = context.WithValue(asyncCtx, "traceID", traceID)
        asyncCtx, cancel := context.WithTimeout(asyncCtx, 30*time.Second)
        defer cancel()
        
        s.sendNotification(asyncCtx, result)
    }()
    
    return nil
}
```

#### Pattern 3: Detached Context Helper
```go
// Helper for creating detached contexts that preserve metadata
func NewDetachedContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
    ctx := context.Background()
    
    // Preserve important values
    if val := parent.Value("traceID"); val != nil {
        ctx = context.WithValue(ctx, "traceID", val)
    }
    if val := parent.Value("userID"); val != nil {
        ctx = context.WithValue(ctx, "userID", val)
    }
    if val := parent.Value("correlationID"); val != nil {
        ctx = context.WithValue(ctx, "correlationID", val)
    }
    
    return context.WithTimeout(ctx, timeout)
}

// Usage
func (s *Service) handleRequest(ctx context.Context) error {
    result := s.processRequest(ctx)
    
    go func() {
        asyncCtx, cancel := NewDetachedContext(ctx, 30*time.Second)
        defer cancel()
        
        s.performBackgroundTask(asyncCtx, result)
    }()
    
    return nil
}
```

#### Pattern 4: Work Queue for High Volume
```go
type AsyncTask struct {
    Type     string
    Data     interface{}
    Metadata map[string]interface{}
}

type TaskProcessor struct {
    queue  chan AsyncTask
    ctx    context.Context
    cancel context.CancelFunc
}

func (tp *TaskProcessor) Start() {
    tp.ctx, tp.cancel = context.WithCancel(context.Background())
    
    go func() {
        for {
            select {
            case task := <-tp.queue:
                tp.processTask(task)
            case <-tp.ctx.Done():
                return
            }
        }
    }()
}

func (tp *TaskProcessor) processTask(task AsyncTask) {
    // Each task gets independent context
    ctx := context.Background()
    for key, value := range task.Metadata {
        ctx = context.WithValue(ctx, key, value)
    }
    
    ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
    defer cancel()
    
    // Process task with independent lifecycle
    tp.handleTaskType(ctx, task)
}

// Usage in handlers
func (s *Service) handleRequest(ctx context.Context) error {
    result := s.processRequest(ctx)
    
    // Submit to work queue instead of spawning goroutines
    s.taskProcessor.Submit(AsyncTask{
        Type: "notification",
        Data: result,
        Metadata: map[string]interface{}{
            "userID":        ctx.Value("userID"),
            "correlationID": ctx.Value("correlationID"),
        },
    })
    
    return nil
}
```

### Real-World Example: Task Notification

```go
// ❌ BEFORE: Notification tied to task runner lifecycle
func (r *Runner) sendNotification() {
    req := &grpc.TaskServiceSendNotificationRequest{
        NodeKey: r.svc.GetNodeConfigService().GetNodeKey(),
        TaskId:  r.tid.Hex(),
    }
    
    // Problem: Uses task runner context, can fail during cleanup
    ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
    defer cancel()
    
    taskClient.SendNotification(ctx, req)
}

// ✅ AFTER: Independent context for reliable async notification
func (r *Runner) sendNotification() {
    req := &grpc.TaskServiceSendNotificationRequest{
        NodeKey: r.svc.GetNodeConfigService().GetNodeKey(),
        TaskId:  r.tid.Hex(),
    }
    
    // Use independent context - notification survives task cleanup
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    taskClient.SendNotification(ctx, req)
}

// Called asynchronously after task completion
go r.sendNotification()
```

## 📏 Timeout Guidelines

| Operation Type | Recommended Timeout | Rationale |
|---------------|-------------------|-----------|
| Database queries | 30 seconds | Most queries should complete quickly |
| HTTP API calls | 30 seconds | External services response time |
| gRPC connections | 30 seconds | Network connection establishment |
| File operations | 10 seconds | Local I/O should be fast |
| LLM/AI operations | 2-10 minutes | AI processing can be slow |
| Background tasks | 5-30 minutes | Depends on task complexity |
| Health checks | 5 seconds | Should be fast |
| Authentication | 10 seconds | OAuth flows |

## 🔍 Code Review Checklist

- [ ] Does the function accept `context.Context` as first parameter?
- [ ] Is `context.Background()` only used for initialization/tests?
- [ ] Are HTTP handlers using request context?
- [ ] Do long-running operations have appropriate timeouts?
- [ ] Is context cancellation checked in loops?
- [ ] Are goroutines properly handling context cancellation?
- [ ] Do gRPC calls use the provided context?
- [ ] Are async operations using independent context when appropriate?
- [ ] Do fire-and-forget operations avoid using parent context?
- [ ] Are important context values preserved in async operations?

## 📚 References

- [Go Context Package](https://pkg.go.dev/context)
- [Context Best Practices](https://go.dev/blog/context-and-structs)
- [gRPC Go Context](https://grpc.io/docs/guides/context/)
