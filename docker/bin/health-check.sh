#!/bin/bash

# Universal health check script for both AstianGO Hub master and worker nodes
# This script checks if the node health file exists and indicates healthy status

HEALTH_FILE="/tmp/astiango_hub_health"
MAX_AGE_SECONDS=60

# Check if health file exists
if [ ! -f "$HEALTH_FILE" ]; then
    echo "Health file not found at $HEALTH_FILE"
    exit 1
fi

# Check if file is recent (modified within last 60 seconds)
if [ $(find "$HEALTH_FILE" -mmin -1 | wc -l) -eq 0 ]; then
    echo "Health file is too old (last modified more than 1 minute ago)"
    cat "$HEALTH_FILE" 2>/dev/null || echo "Could not read health file"
    exit 1
fi

# Check health status from file content
HEALTHY=$(grep '"healthy": true' "$HEALTH_FILE" 2>/dev/null)
if [ -z "$HEALTHY" ]; then
    echo "Node is not healthy according to health file:"
    cat "$HEALTH_FILE" 2>/dev/null || echo "Could not read health file"
    exit 1
fi

# Get node type for logging
NODE_TYPE=$(grep '"node_type"' "$HEALTH_FILE" 2>/dev/null | cut -d'"' -f4)
echo "AstianGO Hub $NODE_TYPE node is healthy"
exit 0
