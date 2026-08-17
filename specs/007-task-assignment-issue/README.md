---
status: complete
created: 2025-10-20
tags: [task-system]
priority: medium
---

# Task Assignment Issue - Visual Explanation

## 🔴 Problem Scenario: Why Scheduled Tasks Get Stuck in Pending

### Timeline Diagram

```mermaid
sequenceDiagram
    participant Cron as Schedule Service (Cron)
    participant DB as Database
    participant Node as Worker Node
    participant Master as Master (FetchTask)
    
    Note over Node: T0: Node Healthy<br/>status: online, active: true
    
    Node->>Node: Goes OFFLINE
    Note over Node: T1: Node Offline<br/>status: offline, active: false
    
    Note over Cron: Cron Triggers<br/>(scheduled task)
    
    Cron->>DB: Query: active=true, status=online
    DB-->>Cron: Returns empty array
    
    Cron->>DB: Create Task with wrong node_id
    Note over DB: Task 123<br/>node_id: node_001 (offline)<br/>status: PENDING
    
    Node->>Node: Comes ONLINE
    Note over Node: T2: Node Online Again<br/>status: online, active: true
    
    loop Every 1 second
        Node->>Master: FetchTask(nodeKey: node_001)
        
        Master->>DB: Query 1: node_id=node_001 AND status=pending
        DB-->>Master: No match or wrong match
        
        Master->>DB: Query 2: node_id=NIL AND status=pending
        DB-->>Master: No match (node_id is set)
        
        Master-->>Node: No task available
    end
    
    Note over DB: Task 123: STUCK FOREVER<br/>Cannot be executed
```

---

## 📊 System Architecture: Current Flow

```mermaid
flowchart TB
    subgraph Master["MASTER NODE"]
        Cron["Schedule Service<br/>(Cron Jobs)"]
        SpiderAdmin["Spider Admin Service<br/>(Task Creation)"]
        FetchLogic["FetchTask Logic<br/>(Task Assignment)"]
        
        Cron -->|"Trigger"| SpiderAdmin
        
        subgraph TaskCreation["Task Creation Flow"]
            GetNodes["1️⃣ getNodeIds()<br/>Query: {active:true, enabled:true, status:online}"]
            CreateTasks["2️⃣ scheduleTasks()<br/>for each nodeId:<br/>task.NodeId = nodeId ⚠️<br/>task.Status = PENDING"]
            
            GetNodes -->|"⚠️ SNAPSHOT<br/>可能已过期!"| CreateTasks
        end
        
        SpiderAdmin --> GetNodes
    end
    
    subgraph Worker["🖥️ WORKER NODE"]
        TaskHandler["🔧 Task Handler Service<br/>(Fetches & Runs Tasks)"]
        FetchLoop["🔄 Loop every 1 second:<br/>FetchTask(nodeKey)"]
        
        TaskHandler --> FetchLoop
    end
    
    subgraph Database["💾 DATABASE"]
        NodesTable[("📋 Nodes Table<br/>status: online/offline<br/>active: true/false")]
        TasksTable[("📋 Tasks Table<br/>node_id: xxx<br/>status: pending")]
    end
    
    GetNodes -.->|"Query"| NodesTable
    CreateTasks -->|"Insert"| TasksTable
    
    FetchLoop -->|"gRPC Request"| FetchLogic
    
    subgraph FetchQueries["FetchTask Query Logic"]
        Q1["1️⃣ Query:<br/>node_id = THIS_NODE<br/>status = PENDING"]
        Q2["2️⃣ Query:<br/>node_id = NIL<br/>status = PENDING"]
        Q3["❌ MISSING!<br/>node_id = OFFLINE_NODE<br/>status = PENDING"]
        
        Q1 -->|"Not found"| Q2
        Q2 -->|"Not found"| Q3
        Q3 -->|"🚫"| ReturnEmpty["Return: No task"]
    end
    
    FetchLogic --> Q1
    Q1 -.->|"Query"| TasksTable
    Q2 -.->|"Query"| TasksTable
    Q3 -.->|"🐛 Never executed!"| TasksTable
    
    ReturnEmpty --> FetchLoop
    
    style Q3 fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style CreateTasks fill:#ffe066,stroke:#fab005
    style GetNodes fill:#ffe066,stroke:#fab005
    style ReturnEmpty fill:#ff6b6b,stroke:#c92a2a,color:#fff
```

---

## 🔍 The Bug in Detail

### Scenario: Task Gets Orphaned

```mermaid
stateDiagram-v2
    [*] --> NodeHealthy
    
    state "Node Healthy T0" as NodeHealthy {
        [*] --> Online
        Online: status online
        Online: active true
        Online: enabled true
    }
    
    NodeHealthy --> NodeOffline
    note right of NodeOffline
        Node crashes or
        network issue
    end note
    
    state "Node Offline T1" as NodeOffline {
        [*] --> Offline
        Offline: status offline
        Offline: active false
        Offline: enabled true
    }
    
    state "Cron Triggers T1" as CronTrigger {
        QueryNodes: Query active true and status online
        QueryNodes --> NoNodesFound
        NoNodesFound: Returns empty array
        NoNodesFound --> TaskCreated
        TaskCreated: BUG Task with stale node_id
    }
    
    NodeOffline --> CronTrigger
    note right of CronTrigger
        Scheduled time arrives
    end note
    
    CronTrigger --> DatabaseT1
    
    state "Database at T1" as DatabaseT1 {
        state "Tasks Table" as TasksT1
        TasksT1: task_123
        TasksT1: node_id node_001 offline
        TasksT1: status PENDING
    }
    
    NodeOffline --> NodeReconnect
    note right of NodeReconnect
        Network restored
    end note
    
    state "Node Reconnect T2" as NodeReconnect {
        [*] --> OnlineAgain
        OnlineAgain: status online
        OnlineAgain: active true
        OnlineAgain: enabled true
    }
    
    NodeReconnect --> FetchAttempt
    
    state "FetchTask Attempt T3" as FetchAttempt {
        Query1: Query 1 node_id equals node_001
        Query1 --> Query1Result
        Query1Result: No match or wrong match
        Query1Result --> Query2
        Query2: Query 2 node_id is NIL
        Query2 --> Query2Result
        Query2Result: No match node_id is set
        Query2Result --> NoTaskReturned
        NoTaskReturned: Return empty
    }
    
    FetchAttempt --> TaskStuck
    
    state "Task Stuck Forever" as TaskStuck {
        [*] --> StuckState
        StuckState: Task 123
        StuckState: status PENDING forever
        StuckState: Never assigned to worker
        StuckState: Never executed
    }
    
    TaskStuck --> [*]
    note left of TaskStuck
        Manual intervention
        required
    end note
```

---

## 🐛 Three Critical Bugs

### Bug #1: Stale Node Snapshot

```mermaid
sequenceDiagram
    participant Sched as Schedule Service
    participant DB as Database
    participant Node1 as Node 001
    
    Note over Node1: ❌ Node 001 goes offline
    
    Sched->>DB: getNodeIds()<br/>Query: {status: online}
    DB-->>Sched: ⚠️ Returns: [node_002]<br/>(Node 001 is offline)
    
    Sched->>DB: Create Task #123<br/>node_id: node_002
    Note over DB: Task assigned to node_002
    
    Note over Node1: ✅ Node 001 comes back online
    
    loop Fetch attempts
        Node1->>DB: Query: node_id=node_001 AND status=pending
        DB-->>Node1: ❌ No match (task has node_002)
        
        Node1->>DB: Query: node_id=NIL AND status=pending
        DB-->>Node1: ❌ No match (task has node_002)
    end
    
    Note over Node1,DB: 🚫 Task never fetched!<br/>⏳ STUCK FOREVER!
```

### Bug #2: Missing Orphaned Task Detection

```mermaid
graph TD
    Start[FetchTask Logic] --> Q1{Query 1:<br/>node_id = THIS_NODE<br/>status = PENDING}
    
    Q1 -->|✅ Found| Return1[Assign & Return Task]
    Q1 -->|❌ Not Found| Q2{Query 2:<br/>node_id = NIL<br/>status = PENDING}
    
    Q2 -->|✅ Found| Return2[Assign & Return Task]
    Q2 -->|❌ Not Found| Missing[❌ MISSING!<br/>Query 3:<br/>node_id = OFFLINE_NODE<br/>status = PENDING]
    
    Missing -.->|Should lead to| Return3[Reassign & Return Task]
    Missing -->|Currently| ReturnEmpty[🚫 Return Empty<br/>Task stuck!]
    
    style Q1 fill:#51cf66,stroke:#2f9e44
    style Q2 fill:#51cf66,stroke:#2f9e44
    style Missing fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style ReturnEmpty fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style Return3 fill:#a9e34b,stroke:#5c940d,stroke-dasharray: 5 5
```

### Bug #3: No Pending Task Reassignment

```mermaid
graph LR
    subgraph Current["❌ Current HandleNodeReconnection()"]
        A1[Node Reconnects] --> B1[Reconcile DISCONNECTED tasks ✅]
        B1 --> C1[Reconcile RUNNING tasks ✅]
        C1 --> D1[❌ MISSING: Reconcile PENDING tasks]
        D1 -.->|Not implemented| E1[Tasks never started remain stuck]
    end
    
    subgraph Needed["✅ Should Include"]
        A2[Node Reconnects] --> B2[Reconcile DISCONNECTED tasks ✅]
        B2 --> C2[Reconcile RUNNING tasks ✅]
        C2 --> D2[✨ NEW: Reconcile PENDING tasks]
        D2 --> E2[Check if assigned node is still valid]
        E2 -->|Node offline| F2[Set node_id = NIL]
        E2 -->|Node online| G2[Keep assignment]
    end
    
    style D1 fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style E1 fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style D2 fill:#a9e34b,stroke:#5c940d
    style F2 fill:#a9e34b,stroke:#5c940d
```

---

## ✅ Solution Visualization

### Fix #1: Enhanced FetchTask Logic

```mermaid
graph TD
    subgraph Before["❌ BEFORE - Tasks get stuck"]
        B1[FetchTask Request] --> BQ1[Query 1: my node_id]
        BQ1 -->|Not found| BQ2[Query 2: nil node_id]
        BQ2 -->|Not found| BR[🚫 Return empty<br/>Task stuck!]
    end
    
    subgraph After["✅ AFTER - Orphaned tasks recovered"]
        A1[FetchTask Request] --> AQ1[Query 1: my node_id]
        AQ1 -->|Not found| AQ2[Query 2: nil node_id]
        AQ2 -->|Not found| AQ3[✨ NEW Query 3:<br/>offline node_ids]
        AQ3 -->|Found| AR[Reassign to me<br/>& return task ✅]
    end
    
    style BR fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style AQ3 fill:#a9e34b,stroke:#5c940d
    style AR fill:#51cf66,stroke:#2f9e44
```

### Fix #2: Pending Task Reassignment

```mermaid
flowchart TD
    Start[Node Reconnects] --> Step1[1. Reconcile DISCONNECTED tasks ✅]
    Step1 --> Step2[2. Reconcile RUNNING tasks ✅]
    Step2 --> Step3[✨ NEW: 3. Check PENDING tasks assigned to me]
    
    Step3 --> Query[Get all pending tasks<br/>with node_id = THIS_NODE]
    Query --> Check{For each task:<br/>Am I really online?}
    
    Check -->|YES: Online & Active| Keep[Keep assignment ✅<br/>Task will be fetched normally]
    Check -->|NO: Offline or Disabled| Reassign[Set node_id = NIL ✨<br/>Allow re-assignment]
    
    Keep --> CheckAge{Task age > 5 min?}
    CheckAge -->|YES| ForceReassign[Force reassignment<br/>for stuck tasks]
    CheckAge -->|NO| Wait[Keep waiting]
    
    Reassign --> Done[✅ Task can be<br/>fetched by any node]
    ForceReassign --> Done
    Wait --> Done
    
    style Step3 fill:#a9e34b,stroke:#5c940d
    style Reassign fill:#a9e34b,stroke:#5c940d
    style Done fill:#51cf66,stroke:#2f9e44
```

### Fix #3: Periodic Cleanup

```mermaid
sequenceDiagram
    participant Timer as ⏰ Timer<br/>(Every 5 min)
    participant Cleanup as 🧹 Cleanup Service
    participant DB as 💾 Database
    
    loop Every 5 minutes
        Timer->>Cleanup: Trigger cleanup
        
        Cleanup->>DB: Find pending tasks > 5 min old
        DB-->>Cleanup: Return aged pending tasks
        
        loop For each task
            Cleanup->>DB: Get node for task.node_id
            
            alt Node is online & active
                DB-->>Cleanup: ✅ Node healthy
                Cleanup->>Cleanup: Keep assignment
            else Node is offline or not found
                DB-->>Cleanup: ❌ Node offline/missing
                Cleanup->>DB: ✨ Update task:<br/>SET node_id = NIL
                Note over DB: Task can now be<br/>fetched by any node!
            end
        end
        
        Cleanup-->>Timer: ✅ Cleanup complete
    end
```

---

## 🎯 Summary

### The Core Problem

```mermaid
graph LR
    A[⏰ 定时任务触发] --> B{节点状态?}
    B -->|✅ Online| C[✅ 正常创建任务<br/>正确的 node_id]
    B -->|❌ Offline| D[🐛 创建任务<br/>错误的 node_id]
    
    C --> E[任务正常执行 ✅]
    
    D --> F[节点重新上线]
    F --> G[📥 FetchTask 尝试获取任务]
    G --> H{能查询到吗?}
    H -->|Query 1| I[❌ node_id 不匹配]
    H -->|Query 2| J[❌ node_id 不是 NIL]
    I --> K[🚫 返回空]
    J --> K
    K --> L[⏳ 任务永远待定!]
    
    style D fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style L fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style E fill:#51cf66,stroke:#2f9e44
```

### Why It Happens

```mermaid
mindmap
  root((🐛 Root Cause))
    Bug 1
      Task Creation
        Uses snapshot of nodes
        可能已过期
        Assigns wrong node_id
    Bug 2
      FetchTask Logic
        Only 2 queries
          My node_id ✅
          NIL node_id ✅
        Missing 3rd query
          Offline node_id ❌
    Bug 3
      Node Reconnection
        Handles running tasks ✅
        Handles disconnected tasks ✅
        Missing pending tasks ❌
```

### The Fix

```mermaid
graph TB
    Problem[🐛 Problem:<br/>Orphaned Tasks] --> Solution[💡 Solution:<br/>Detect & Reassign]
    
    Solution --> Fix1[Fix 1: Enhanced FetchTask<br/>Add offline node query]
    Solution --> Fix2[Fix 2: Node Reconnection<br/>Reassign pending tasks]
    Solution --> Fix3[Fix 3: Periodic Cleanup<br/>Reset stale assignments]
    
    Fix1 --> Result[✅ Tasks can be<br/>fetched again]
    Fix2 --> Result
    Fix3 --> Result
    
    Result --> Success[🎉 No more stuck tasks!<br/>定时任务正常执行]
    
    style Problem fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style Solution fill:#fab005,stroke:#e67700
    style Fix1 fill:#a9e34b,stroke:#5c940d
    style Fix2 fill:#a9e34b,stroke:#5c940d
    style Fix3 fill:#a9e34b,stroke:#5c940d
    style Success fill:#51cf66,stroke:#2f9e44
```

---

## 📚 Key Takeaways

| Issue | Current Behavior | Expected Behavior | Priority |
|-------|-----------------|-------------------|----------|
| **Orphaned Tasks** | Tasks assigned to offline nodes never get fetched | FetchTask should detect and reassign them | 🔴 **HIGH** |
| **Stale Assignments** | node_id set at creation time, never updated | Should be validated/updated on node status change | 🟡 **MEDIUM** |
| **No Cleanup** | Old pending tasks accumulate forever | Periodic cleanup should reset stale assignments | 🟡 **MEDIUM** |

---

**Generated**: 2025-10-19  
**File**: `/tmp/task_assignment_issue_diagram.md`  
**Status**: Ready for implementation 🚀
