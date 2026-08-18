import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { requireDestructiveConfirmation } from '../security.js';
import { z } from "zod";

const SCHEDULE_TOOLS = {
  list_schedules: "astiango_hub_list_schedules",
  get_schedule: "astiango_hub_get_schedule",
  create_schedule: "astiango_hub_create_schedule",
  update_schedule: "astiango_hub_update_schedule",
  delete_schedule: "astiango_hub_delete_schedule",
  enable_schedule: "astiango_hub_enable_schedule",
  disable_schedule: "astiango_hub_disable_schedule",
};

export function configureScheduleTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    SCHEDULE_TOOLS.list_schedules,
    "List all schedules in AstianGO Hub",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of schedules per page (default: 10)"),
    },
    async ({ page, size }) => {
      try {
        const response = await client.getSchedules({ page, size });
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
              text: `Error listing schedules: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.get_schedule,
    "Get details of a specific schedule",
    {
      schedule_id: z.string().describe("The ID of the schedule to retrieve"),
    },
    async ({ schedule_id }) => {
      try {
        const response = await client.getSchedule(schedule_id);
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
              text: `Error getting schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.create_schedule,
    "Create a new schedule",
    {
      name: z.string().describe("Name of the schedule"),
      description: z.string().optional().describe("Description of the schedule"),
      spider_id: z.string().describe("ID of the spider to schedule"),
      cron: z.string().describe("Cron expression for the schedule (e.g., '0 0 * * *' for daily at midnight)"),
      cmd: z.string().optional().describe("Command to override for scheduled runs"),
      param: z.string().optional().describe("Parameters to override for scheduled runs"),
      mode: z.enum(["random", "all", "selected-nodes"]).optional().describe("Task execution mode"),
      node_ids: z.array(z.string()).optional().describe("Node IDs for selected-nodes mode"),
      priority: z.number().min(1).max(10).optional().describe("Task priority (1-10)"),
      enabled: z.boolean().optional().describe("Whether the schedule is enabled (default: true)"),
    },
    async ({ name, description, spider_id, cron, cmd, param, mode, node_ids, priority, enabled = true }) => {
      try {
        const scheduleData = {
          name,
          description,
          spider_id,
          cron,
          cmd,
          param,
          mode,
          node_ids,
          priority,
          enabled,
        };
        const response = await client.createSchedule(scheduleData);
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
              text: `Error creating schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.update_schedule,
    "Update an existing schedule",
    {
      schedule_id: z.string().describe("The ID of the schedule to update"),
      name: z.string().optional().describe("New name for the schedule"),
      description: z.string().optional().describe("New description for the schedule"),
      spider_id: z.string().optional().describe("New spider ID for the schedule"),
      cron: z.string().optional().describe("New cron expression for the schedule"),
      cmd: z.string().optional().describe("New command for the schedule"),
      param: z.string().optional().describe("New parameters for the schedule"),
      mode: z.enum(["random", "all", "selected-nodes"]).optional().describe("New task execution mode"),
      node_ids: z.array(z.string()).optional().describe("New node IDs for selected-nodes mode"),
      priority: z.number().min(1).max(10).optional().describe("New task priority (1-10)"),
      enabled: z.boolean().optional().describe("Whether the schedule is enabled"),
    },
    async ({ schedule_id, name, description, spider_id, cron, cmd, param, mode, node_ids, priority, enabled }) => {
      try {
        const updateData = {
          ...(name && { name }),
          ...(description && { description }),
          ...(spider_id && { spider_id }),
          ...(cron && { cron }),
          ...(cmd && { cmd }),
          ...(param && { param }),
          ...(mode && { mode }),
          ...(node_ids && { node_ids }),
          ...(priority && { priority }),
          ...(enabled !== undefined && { enabled }),
        };
        const response = await client.updateSchedule(schedule_id, updateData);
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
              text: `Error updating schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.delete_schedule,
    "Delete a schedule",
    {
      schedule_id: z.string().describe("The ID of the schedule to delete"),
      confirmation: z.string().describe('Required: DELETE:<schedule_id>'),
    },
    async ({ schedule_id, confirmation }) => {
      try {
        requireDestructiveConfirmation(confirmation, 'DELETE', schedule_id);
        await client.deleteSchedule(schedule_id);
        return {
          content: [
            {
              type: "text",
              text: `Schedule ${schedule_id} deleted successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error deleting schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.enable_schedule,
    "Enable a schedule",
    {
      schedule_id: z.string().describe("The ID of the schedule to enable"),
    },
    async ({ schedule_id }) => {
      try {
        await client.enableSchedule(schedule_id);
        return {
          content: [
            {
              type: "text",
              text: `Schedule ${schedule_id} enabled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error enabling schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SCHEDULE_TOOLS.disable_schedule,
    "Disable a schedule",
    {
      schedule_id: z.string().describe("The ID of the schedule to disable"),
      confirmation: z.string().describe('Required: DISABLE:<schedule_id>'),
    },
    async ({ schedule_id, confirmation }) => {
      try {
        requireDestructiveConfirmation(confirmation, 'DISABLE', schedule_id);
        await client.disableSchedule(schedule_id);
        return {
          content: [
            {
              type: "text",
              text: `Schedule ${schedule_id} disabled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error disabling schedule: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { SCHEDULE_TOOLS };
