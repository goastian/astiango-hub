import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { z } from "zod";

const TASK_TOOLS = {
  list_tasks: "astiango_hub_list_tasks",
  get_task: "astiango_hub_get_task",
  cancel_task: "astiango_hub_cancel_task",
  restart_task: "astiango_hub_restart_task",
  delete_task: "astiango_hub_delete_task",
  get_task_logs: "astiango_hub_get_task_logs",
  get_task_results: "astiango_hub_get_task_results",
};

export function configureTaskTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    TASK_TOOLS.list_tasks,
    "List tasks in AstianGO Hub",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of tasks per page (default: 10)"),
      spider_id: z.string().optional().describe("Filter by spider ID"),
      status: z.string().optional().describe("Filter by task status (pending, assigned, running, finished, error, cancelled, abnormal)"),
    },
    async ({ page, size, spider_id, status }) => {
      try {
        const response = await client.getTasks({ page, size, spider_id, status });
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(response, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error listing tasks: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.get_task,
    "Get details of a specific task",
    {
      task_id: z.string().describe("The ID of the task to retrieve"),
    },
    async ({ task_id }) => {
      try {
        const response = await client.getTask(task_id);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(response, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting task: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.cancel_task,
    "Cancel a running task",
    {
      task_id: z.string().describe("The ID of the task to cancel"),
    },
    async ({ task_id }) => {
      try {
        await client.cancelTask(task_id);
        return {
          content: [
            {
              type: "text",
              text: `Task ${task_id} cancelled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error cancelling task: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.restart_task,
    "Restart a task",
    {
      task_id: z.string().describe("The ID of the task to restart"),
    },
    async ({ task_id }) => {
      try {
        const response = await client.restartTask(task_id);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(response, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error restarting task: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.delete_task,
    "Delete a task",
    {
      task_id: z.string().describe("The ID of the task to delete"),
    },
    async ({ task_id }) => {
      try {
        await client.deleteTask(task_id);
        return {
          content: [
            {
              type: "text",
              text: `Task ${task_id} deleted successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error deleting task: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.get_task_logs,
    "Get logs from a task",
    {
      task_id: z.string().describe("The ID of the task"),
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of log lines per page (default: 100)"),
    },
    async ({ task_id, page, size }) => {
      try {
        const response = await client.getTaskLogs(task_id, { page, size });
        return {
          content: [
            {
              type: "text",
              text: Array.isArray(response.data) 
                ? response.data.join('\n') 
                : JSON.stringify(response, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting task logs: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    TASK_TOOLS.get_task_results,
    "Get results data from a task",
    {
      task_id: z.string().describe("The ID of the task"),
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of results per page (default: 10)"),
    },
    async ({ task_id, page, size }) => {
      try {
        const response = await client.getTaskResults(task_id, { page, size });
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(response, null, 2),
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting task results: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { TASK_TOOLS };
