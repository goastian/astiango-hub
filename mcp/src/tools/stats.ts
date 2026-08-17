import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { z } from "zod";

const STATS_TOOLS = {
  get_spider_stats: "astiango_hub_get_spider_stats",
  get_task_stats: "astiango_hub_get_task_stats",
};

export function configureStatsTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    STATS_TOOLS.get_spider_stats,
    "Get statistics for a specific spider",
    {
      spider_id: z.string().describe("The ID of the spider to get statistics for"),
    },
    async ({ spider_id }) => {
      try {
        const response = await client.getSpiderStats(spider_id);
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
              text: `Error getting spider statistics: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    STATS_TOOLS.get_task_stats,
    "Get statistics for a specific task",
    {
      task_id: z.string().describe("The ID of the task to get statistics for"),
    },
    async ({ task_id }) => {
      try {
        const response = await client.getTaskStats(task_id);
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
              text: `Error getting task statistics: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { STATS_TOOLS };
