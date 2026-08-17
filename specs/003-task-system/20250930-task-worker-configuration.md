# AstianGO Hub Task Worker Configuration Examples

## Basic Configuration

### config.yml
```yaml
# Task execution configuration
task:
  workers: 20  # Number of concurrent task workers (default: 10)

# Node configuration (optional)
node:
  maxRunners: 50  # Maximum total tasks per node (0 = unlimited)
```

### Environment Variables
```bash
# Set via environment variables
export ASTIANGO_TASK_WORKERS=20
export ASTIANGO_NODE_MAXRUNNERS=50
```

## Configuration Guidelines

### Worker Count Recommendations

| Scenario | Task Workers | Queue Size | Memory Usage |
|----------|-------------|------------|--------------|
| Development | 5-10 | 25-50 | ~100MB |
| Small Production | 15-20 | 75-100 | ~200MB |
| Medium Production | 25-35 | 125-175 | ~400MB |
| Large Production | 40-60 | 200-300 | ~800MB |

### Factors to Consider

1. **Task Complexity**: CPU/Memory intensive tasks need fewer workers
2. **Task Duration**: Long-running tasks need more workers for throughput
3. **System Resources**: Balance workers with available CPU/Memory
4. **Database Load**: More workers = more database connections
5. **External Dependencies**: Network-bound tasks can handle more workers

### Performance Tuning

#### Too Few Workers (Queue Full Errors)
```log
WARN task queue is full (50/50), consider increasing task.workers configuration
```
**Solution**: Increase `task.workers` value

#### Too Many Workers (Resource Exhaustion)
```log
ERROR failed to create task runner: out of memory
ERROR database connection pool exhausted
```
**Solution**: Decrease `task.workers` value

#### Optimal Configuration
```log
INFO Task handler service started with 20 workers and queue size 100
DEBUG task[abc123] queued, queue usage: 5/100
```

## Docker Configuration

### docker-compose.yml
```yaml
version: '3'
services:
  astiango-hub-master:
    image: goastian/astiango-hub:latest
    environment:
      - ASTIANGO_TASK_WORKERS=25
      - ASTIANGO_NODE_MAXRUNNERS=100
    # ... other config

  astiango-hub-worker:
    image: goastian/astiango-hub:latest
    environment:
      - ASTIANGO_TASK_WORKERS=30  # Workers can be different per node
      - ASTIANGO_NODE_MAXRUNNERS=150
    # ... other config
```

### Kubernetes ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: astiango-hub-config
data:
  config.yml: |
    task:
      workers: 25
    node:
      maxRunners: 100
```

## Monitoring Worker Performance

### Log Monitoring
```bash
# Monitor worker pool status
grep -E "(workers|queue usage|queue is full)" /var/log/astiango-hub/astiango-hub.log

# Monitor task throughput
grep -E "(task.*queued|task.*finished)" /var/log/astiango-hub/astiango-hub.log | wc -l
```

### Metrics to Track
- Queue utilization percentage
- Average task execution time
- Worker pool saturation
- Memory usage per worker
- Task success/failure rates

## Troubleshooting

### Queue Always Full
1. Increase worker count: `task.workers`
2. Check task complexity and optimization
3. Verify database performance
4. Consider scaling horizontally (more nodes)

### High Memory Usage
1. Decrease worker count
2. Optimize task memory usage
3. Implement task batching
4. Add memory monitoring alerts

### Slow Task Processing
1. Profile individual tasks
2. Check database query performance
3. Optimize external API calls
4. Consider async task patterns

## Testing Configuration Changes

```bash
# Test new configuration
export ASTIANGO_TASK_WORKERS=30
./scripts/test_goroutine_fixes.sh 900 10

# Monitor during peak load
./scripts/test_goroutine_fixes.sh 3600 5
```

## Best Practices

1. **Start Conservative**: Begin with default values and monitor
2. **Load Test**: Always test configuration changes under load
3. **Monitor Metrics**: Track queue utilization and task throughput
4. **Scale Gradually**: Increase worker count in small increments
5. **Resource Limits**: Set appropriate memory/CPU limits in containers
6. **High Availability**: Configure different worker counts per node type
