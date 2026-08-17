import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { z } from "zod";

const SPIDER_TOOLS = {
  list_spiders: "astiango_hub_list_spiders",
  get_spider: "astiango_hub_get_spider",
  create_spider: "astiango_hub_create_spider",
  update_spider: "astiango_hub_update_spider",
  delete_spider: "astiango_hub_delete_spider",
  run_spider: "astiango_hub_run_spider",
  list_spider_files: "astiango_hub_list_spider_files",
  get_spider_file_content: "astiango_hub_get_spider_file_content",
  save_spider_file: "astiango_hub_save_spider_file",
};

export function configureSpiderTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    SPIDER_TOOLS.list_spiders,
    "List all spiders in AstianGO Hub",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of spiders per page (default: 10)"),
    },
    async ({ page, size }) => {
      try {
        const response = await client.getSpiders({ page, size });
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
              text: `Error listing spiders: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.get_spider,
    "Get details of a specific spider",
    {
      spider_id: z.string().describe("The ID of the spider to retrieve"),
    },
    async ({ spider_id }) => {
      try {
        const response = await client.getSpider(spider_id);
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
              text: `Error getting spider: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.create_spider,
    "Create a new spider",
    {
      name: z.string().describe("Name of the spider"),
      description: z.string().optional().describe("Description of the spider"),
      cmd: z.string().describe("Command to execute the spider"),
      param: z.string().optional().describe("Parameters for the spider command"),
      project_id: z.string().optional().describe("Project ID to associate with the spider"),
      database_id: z.string().optional().describe("Database ID for data storage"),
      col_name: z.string().optional().describe("Collection/table name for results"),
      db_name: z.string().optional().describe("Database name for results"),
      mode: z.enum(["random", "all", "selected-nodes"]).optional().describe("Task execution mode"),
      node_ids: z.array(z.string()).optional().describe("Node IDs for selected-nodes mode"),
      git_id: z.string().optional().describe("Git repository ID"),
      git_root_path: z.string().optional().describe("Git root path"),
      template: z.string().optional().describe("Spider template"),
      priority: z.number().min(1).max(10).optional().describe("Priority (1-10, default: 5)"),
    },
    async ({ name, description, cmd, param, project_id, database_id, col_name, db_name, mode, node_ids, git_id, git_root_path, template, priority }) => {
      try {
        const spiderData = {
          name,
          description,
          cmd,
          param,
          project_id,
          database_id,
          col_name,
          db_name,
          mode,
          node_ids,
          git_id,
          git_root_path,
          template,
          priority: priority || 5,
        };
        const response = await client.createSpider(spiderData);
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
              text: `Error creating spider: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.update_spider,
    "Update an existing spider",
    {
      spider_id: z.string().describe("The ID of the spider to update"),
      name: z.string().optional().describe("New name for the spider"),
      description: z.string().optional().describe("New description for the spider"),
      cmd: z.string().optional().describe("New command to execute the spider"),
      param: z.string().optional().describe("New parameters for the spider command"),
      project_id: z.string().optional().describe("New project ID to associate with the spider"),
      database_id: z.string().optional().describe("New database ID for data storage"),
      col_name: z.string().optional().describe("New collection/table name for results"),
      db_name: z.string().optional().describe("New database name for results"),
      mode: z.enum(["random", "all", "selected-nodes"]).optional().describe("New task execution mode"),
      node_ids: z.array(z.string()).optional().describe("New node IDs for selected-nodes mode"),
      git_id: z.string().optional().describe("New git repository ID"),
      git_root_path: z.string().optional().describe("New git root path"),
      template: z.string().optional().describe("New spider template"),
      priority: z.number().min(1).max(10).optional().describe("New priority (1-10)"),
    },
    async ({ spider_id, name, description, cmd, param, project_id, database_id, col_name, db_name, mode, node_ids, git_id, git_root_path, template, priority }) => {
      try {
        const updateData = {
          ...(name && { name }),
          ...(description && { description }),
          ...(cmd && { cmd }),
          ...(param && { param }),
          ...(project_id && { project_id }),
          ...(database_id && { database_id }),
          ...(col_name && { col_name }),
          ...(db_name && { db_name }),
          ...(mode && { mode }),
          ...(node_ids && { node_ids }),
          ...(git_id && { git_id }),
          ...(git_root_path && { git_root_path }),
          ...(template && { template }),
          ...(priority && { priority }),
        };
        const response = await client.updateSpider(spider_id, updateData);
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
              text: `Error updating spider: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.delete_spider,
    "Delete a spider",
    {
      spider_id: z.string().describe("The ID of the spider to delete"),
    },
    async ({ spider_id }) => {
      try {
        await client.deleteSpider(spider_id);
        return {
          content: [
            {
              type: "text",
              text: `Spider ${spider_id} deleted successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error deleting spider: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.run_spider,
    "Run a spider",
    {
      spider_id: z.string().describe("The ID of the spider to run"),
      cmd: z.string().optional().describe("Override command for this run"),
      param: z.string().optional().describe("Override parameters for this run"),
      priority: z.number().min(1).max(10).optional().describe("Task priority (1-10)"),
      mode: z.enum(["random", "all", "selected-nodes"]).optional().describe("Task execution mode"),
      node_ids: z.array(z.string()).optional().describe("Node IDs for selected-nodes mode"),
    },
    async ({ spider_id, cmd, param, priority, mode, node_ids }) => {
      try {
        const runData = {
          ...(cmd && { cmd }),
          ...(param && { param }),
          ...(priority && { priority }),
          ...(mode && { mode }),
          ...(node_ids && { node_ids }),
        };
        const response = await client.runSpider(spider_id, runData);
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
              text: `Error running spider: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.list_spider_files,
    "List files in a spider directory",
    {
      spider_id: z.string().describe("The ID of the spider"),
      path: z.string().optional().describe("Path within the spider directory (default: root)"),
    },
    async ({ spider_id, path }) => {
      try {
        const response = await client.getSpiderFiles(spider_id, path);
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
              text: `Error listing spider files: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.get_spider_file_content,
    "Get the content of a spider file",
    {
      spider_id: z.string().describe("The ID of the spider"),
      file_path: z.string().describe("Path to the file within the spider directory"),
    },
    async ({ spider_id, file_path }) => {
      try {
        const response = await client.getSpiderFileContent(spider_id, file_path);
        return {
          content: [
            {
              type: "text",
              text: response.data || "File is empty",
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error getting file content: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    SPIDER_TOOLS.save_spider_file,
    "Save content to a spider file",
    {
      spider_id: z.string().describe("The ID of the spider"),
      file_path: z.string().describe("Path to the file within the spider directory"),
      content: z.string().describe("Content to save to the file"),
    },
    async ({ spider_id, file_path, content }) => {
      try {
        await client.saveSpiderFile(spider_id, file_path, content);
        return {
          content: [
            {
              type: "text",
              text: `File ${file_path} saved successfully for spider ${spider_id}.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error saving file: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { SPIDER_TOOLS };
