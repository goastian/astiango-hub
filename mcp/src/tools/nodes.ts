import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client";
import { z } from "zod";

const NODE_TOOLS = {
  list_nodes: "astiango_hub_list_nodes",
  get_node: "astiango_hub_get_node",
  update_node: "astiango_hub_update_node",
  enable_node: "astiango_hub_enable_node",
  disable_node: "astiango_hub_disable_node",
};

export function configureNodeTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    NODE_TOOLS.list_nodes,
    "List all nodes in AstianGO Hub cluster",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of nodes per page (default: 10)"),
    },
    async ({ page, size }) => {
      try {
        const response = await client.getNodes({ page, size });
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
              text: `Error listing nodes: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    NODE_TOOLS.get_node,
    "Get details of a specific node",
    {
      node_id: z.string().describe("The ID of the node to retrieve"),
    },
    async ({ node_id }) => {
      try {
        const response = await client.getNode(node_id);
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
              text: `Error getting node: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    NODE_TOOLS.update_node,
    "Update a node configuration",
    {
      node_id: z.string().describe("The ID of the node to update"),
      name: z.string().optional().describe("New name for the node"),
      description: z.string().optional().describe("New description for the node"),
      max_runners: z.number().optional().describe("New maximum number of concurrent runners"),
      enabled: z.boolean().optional().describe("Whether the node is enabled"),
    },
    async ({ node_id, name, description, max_runners, enabled }) => {
      try {
        const updateData = {
          ...(name && { name }),
          ...(description && { description }),
          ...(max_runners && { max_runners }),
          ...(enabled !== undefined && { enabled }),
        };
        const response = await client.updateNode(node_id, updateData);
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
              text: `Error updating node: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    NODE_TOOLS.enable_node,
    "Enable a node",
    {
      node_id: z.string().describe("The ID of the node to enable"),
    },
    async ({ node_id }) => {
      try {
        await client.enableNode(node_id);
        return {
          content: [
            {
              type: "text",
              text: `Node ${node_id} enabled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error enabling node: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    NODE_TOOLS.disable_node,
    "Disable a node",
    {
      node_id: z.string().describe("The ID of the node to disable"),
    },
    async ({ node_id }) => {
      try {
        await client.disableNode(node_id);
        return {
          content: [
            {
              type: "text",
              text: `Node ${node_id} disabled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error disabling node: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { NODE_TOOLS };
