# AstianGO Hub MCP Server

A Model Context Protocol (MCP) server for interacting with [AstianGO Hub](https://github.com/goastian/astiango-hub), a distributed web crawler management platform. This server provides tools to manage spiders, tasks, schedules, and monitor your AstianGO Hub cluster through an AI assistant.

This package is part of an independent fork. Upstream attribution is retained in its LICENSE and in the repository NOTICE.

## Features

### Spider Management
- List, create, update, and delete spiders
- Run spiders with custom parameters
- Browse and edit spider files
- View spider execution history

### Task Management
- Monitor running and completed tasks
- Cancel, restart, and delete tasks
- View task logs and results
- Filter tasks by spider, status, or time range

### Schedule Management
- Create and manage cron-based schedules
- Enable/disable schedules
- View scheduled task history

### Node Monitoring
- List cluster nodes and their status
- Monitor node health and availability

### System Monitoring
- Health checks and system status
- Comprehensive cluster overview

## Installation

```bash
npm install
npm run build
```

## Usage

### Basic Usage

```bash
# Start the MCP server
mcp-server-astiango-hub <astiango_hub_url> [api_token]

# Examples:
mcp-server-astiango-hub http://localhost:8080
mcp-server-astiango-hub https://astiango-hub.example.com your-api-token
```

### Environment Variables

You can also set the API token via environment variable:

```bash
export ASTIANGO_API_TOKEN=your-api-token
mcp-server-astiango-hub http://localhost:8080
```

### With MCP Inspector

For development and testing, you can use the MCP Inspector:

```bash
npm run inspect
```

### Integration with AI Assistants

This MCP server is designed to work with AI assistants that support the Model Context Protocol. Configure your AI assistant to connect to this server to enable AstianGO Hub management capabilities.

## Available Tools

### Spider Tools
- `astiango_hub_list_spiders` - List all spiders with optional pagination
- `astiango_hub_get_spider` - Get detailed information about a specific spider
- `astiango_hub_create_spider` - Create a new spider
- `astiango_hub_update_spider` - Update spider configuration
- `astiango_hub_delete_spider` - Delete a spider
- `astiango_hub_run_spider` - Execute a spider
- `astiango_hub_list_spider_files` - Browse spider files and directories
- `astiango_hub_get_spider_file_content` - Read spider file content
- `astiango_hub_save_spider_file` - Save content to spider files

### Task Tools
- `astiango_hub_list_tasks` - List tasks with filtering options
- `astiango_hub_get_task` - Get detailed task information
- `astiango_hub_cancel_task` - Cancel a running task
- `astiango_hub_restart_task` - Restart a completed or failed task
- `astiango_hub_delete_task` - Delete a task
- `astiango_hub_get_task_logs` - Retrieve task execution logs
- `astiango_hub_get_task_results` - Get data collected by a task

### Schedule Tools
- `astiango_hub_list_schedules` - List all schedules
- `astiango_hub_get_schedule` - Get schedule details
- `astiango_hub_create_schedule` - Create a new cron schedule
- `astiango_hub_update_schedule` - Update schedule configuration
- `astiango_hub_delete_schedule` - Delete a schedule
- `astiango_hub_enable_schedule` - Enable a schedule
- `astiango_hub_disable_schedule` - Disable a schedule

### Node Tools
- `astiango_hub_list_nodes` - List cluster nodes
- `astiango_hub_get_node` - Get node details and status

### System Tools
- `astiango_hub_health_check` - Check system health
- `astiango_hub_system_status` - Get comprehensive system overview

## Available Prompts

The server includes several helpful prompts for common workflows:

### `spider-analysis`
Analyze spider performance and provide optimization insights.

**Parameters:**
- `spider_id` (required) - ID of the spider to analyze
- `time_range` (optional) - Time range for analysis (e.g., '7d', '30d', '90d')

### `task-debugging`
Debug failed tasks and identify root causes.

**Parameters:**
- `task_id` (required) - ID of the failed task

### `spider-setup`
Guide for creating and configuring new spiders.

**Parameters:**
- `spider_name` (required) - Name for the new spider
- `target_website` (optional) - Target website to scrape
- `spider_type` (optional) - Type of spider (scrapy, selenium, custom)

### `system-monitoring`
Monitor system health and performance.

**Parameters:**
- `focus_area` (optional) - Area to focus on (nodes, tasks, storage, overall)

## Example Interactions

### Create and Run a Spider
```
AI: I'll help you create a new spider for scraping news articles.

[Uses astiango_hub_create_spider with appropriate parameters]
[Uses astiango_hub_run_spider to test the spider]
[Uses astiango_hub_get_task_logs to check execution]
```

### Debug a Failed Task
```
User: "My task abc123 failed, can you help me debug it?"

[Uses task-debugging prompt]
[AI retrieves task details, logs, and provides analysis]
```

### Monitor System Health
```
User: "How is my AstianGO Hub cluster performing?"

[Uses system-monitoring prompt]
[AI provides comprehensive health overview and recommendations]
```

## Configuration

### AstianGO Hub Setup

Ensure your AstianGO Hub instance is accessible and optionally configure API authentication:

1. Make sure AstianGO Hub is running and accessible at the specified URL
2. If using authentication, obtain an API token from your AstianGO Hub instance
3. Configure the token via command line argument or environment variable

### MCP Client Configuration

Add this server to your MCP client configuration:

```json
{
  "servers": {
    "astiango-hub": {
      "command": "mcp-server-astiango-hub",
      "args": ["http://localhost:8080", "your-api-token"]
    }
  }
}
```

## Development

### Building
```bash
npm run build
```

### Watching for Changes
```bash
npm run watch
```

### Testing
```bash
npm test
```

### Linting
```bash
npm run lint
npm run lint:fix
```

## Requirements

- Node.js 18+
- A running AstianGO Hub instance
- Valid network access to the AstianGO Hub API

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Check the [AstianGO Hub documentation](https://docs.astiango-hub.cn)
- Review the [MCP specification](https://modelcontextprotocol.io)
- Open an issue in this repository
