import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { z } from "zod";

const GIT_TOOLS = {
  list_git_repos: "astiango_hub_list_git_repos",
  get_git_repo: "astiango_hub_get_git_repo",
  create_git_repo: "astiango_hub_create_git_repo",
  update_git_repo: "astiango_hub_update_git_repo",
  delete_git_repo: "astiango_hub_delete_git_repo",
  clone_git_repo: "astiango_hub_clone_git_repo",
  pull_git_repo: "astiango_hub_pull_git_repo",
};

export function configureGitTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    GIT_TOOLS.list_git_repos,
    "List all Git repositories in AstianGO Hub",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of repositories per page (default: 10)"),
    },
    async ({ page, size }) => {
      try {
        const response = await client.getGitRepos({ page, size });
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
              text: `Error listing Git repositories: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.get_git_repo,
    "Get details of a specific Git repository",
    {
      git_id: z.string().describe("The ID of the Git repository to retrieve"),
    },
    async ({ git_id }) => {
      try {
        const response = await client.getGitRepo(git_id);
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
              text: `Error getting Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.create_git_repo,
    "Create a new Git repository connection",
    {
      name: z.string().describe("Name of the Git repository"),
      url: z.string().describe("Git repository URL"),
      auth_type: z.enum(["public", "private", "token"]).optional().describe("Authentication type"),
      username: z.string().optional().describe("Git username (for private repos)"),
      password: z.string().optional().describe("Git password or token"),
    },
    async ({ name, url, auth_type, username, password }) => {
      try {
        const gitData = {
          name,
          url,
          auth_type,
          username,
          password,
        };
        const response = await client.createGitRepo(gitData);
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
              text: `Error creating Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.update_git_repo,
    "Update an existing Git repository connection",
    {
      git_id: z.string().describe("The ID of the Git repository to update"),
      name: z.string().optional().describe("New name for the repository"),
      url: z.string().optional().describe("New Git repository URL"),
      auth_type: z.enum(["public", "private", "token"]).optional().describe("New authentication type"),
      username: z.string().optional().describe("New Git username"),
      password: z.string().optional().describe("New Git password or token"),
    },
    async ({ git_id, name, url, auth_type, username, password }) => {
      try {
        const updateData = {
          ...(name && { name }),
          ...(url && { url }),
          ...(auth_type && { auth_type }),
          ...(username && { username }),
          ...(password && { password }),
        };
        const response = await client.updateGitRepo(git_id, updateData);
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
              text: `Error updating Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.delete_git_repo,
    "Delete a Git repository connection",
    {
      git_id: z.string().describe("The ID of the Git repository to delete"),
    },
    async ({ git_id }) => {
      try {
        await client.deleteGitRepo(git_id);
        return {
          content: [
            {
              type: "text",
              text: `Git repository ${git_id} deleted successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error deleting Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.clone_git_repo,
    "Clone a Git repository",
    {
      git_id: z.string().describe("The ID of the Git repository to clone"),
    },
    async ({ git_id }) => {
      try {
        await client.cloneGitRepo(git_id);
        return {
          content: [
            {
              type: "text",
              text: `Git repository ${git_id} cloned successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error cloning Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    GIT_TOOLS.pull_git_repo,
    "Pull latest changes from a Git repository",
    {
      git_id: z.string().describe("The ID of the Git repository to pull"),
    },
    async ({ git_id }) => {
      try {
        await client.pullGitRepo(git_id);
        return {
          content: [
            {
              type: "text",
              text: `Git repository ${git_id} pulled successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error pulling Git repository: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { GIT_TOOLS };
