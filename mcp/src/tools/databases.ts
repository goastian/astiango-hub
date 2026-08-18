import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "../client.js";
import { requireDestructiveConfirmation } from '../security.js';
import { z } from "zod";

const DATABASE_TOOLS = {
  list_databases: "astiango_hub_list_databases",
  get_database: "astiango_hub_get_database",
  create_database: "astiango_hub_create_database",
  update_database: "astiango_hub_update_database",
  delete_database: "astiango_hub_delete_database",
  test_database_connection: "astiango_hub_test_database_connection",
};

export function configureDatabaseTools(server: McpServer, client: AstianGoHubClient) {
  server.tool(
    DATABASE_TOOLS.list_databases,
    "List all databases in AstianGO Hub",
    {
      page: z.number().optional().describe("Page number for pagination (default: 1)"),
      size: z.number().optional().describe("Number of databases per page (default: 10)"),
    },
    async ({ page, size }) => {
      try {
        const response = await client.getDatabases({ page, size });
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
              text: `Error listing databases: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    DATABASE_TOOLS.get_database,
    "Get details of a specific database",
    {
      database_id: z.string().describe("The ID of the database to retrieve"),
    },
    async ({ database_id }) => {
      try {
        const response = await client.getDatabase(database_id);
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
              text: `Error getting database: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    DATABASE_TOOLS.create_database,
    "Create a new database connection",
    {
      name: z.string().describe("Name of the database connection"),
      description: z.string().optional().describe("Description of the database"),
      data_source: z.string().describe("Data source type (mongo, mysql, postgres, etc.)"),
      host: z.string().describe("Database host"),
      port: z.number().describe("Database port"),
      database: z.string().optional().describe("Database name"),
      username: z.string().optional().describe("Database username"),
      password: z.string().optional().describe("Database password"),
      uri: z.string().optional().describe("Database URI (alternative to individual connection params)"),
    },
    async ({ name, description, data_source, host, port, database, username, password, uri }) => {
      try {
        const databaseData = {
          name,
          description,
          data_source,
          host,
          port,
          database,
          username,
          password,
          uri,
        };
        const response = await client.createDatabase(databaseData);
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
              text: `Error creating database: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    DATABASE_TOOLS.update_database,
    "Update an existing database connection",
    {
      database_id: z.string().describe("The ID of the database to update"),
      name: z.string().optional().describe("New name for the database"),
      description: z.string().optional().describe("New description for the database"),
      data_source: z.string().optional().describe("New data source type"),
      host: z.string().optional().describe("New database host"),
      port: z.number().optional().describe("New database port"),
      database: z.string().optional().describe("New database name"),
      username: z.string().optional().describe("New database username"),
      password: z.string().optional().describe("New database password"),
      uri: z.string().optional().describe("New database URI"),
    },
    async ({ database_id, name, description, data_source, host, port, database, username, password, uri }) => {
      try {
        const updateData = {
          ...(name && { name }),
          ...(description && { description }),
          ...(data_source && { data_source }),
          ...(host && { host }),
          ...(port && { port }),
          ...(database && { database }),
          ...(username && { username }),
          ...(password && { password }),
          ...(uri && { uri }),
        };
        const response = await client.updateDatabase(database_id, updateData);
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
              text: `Error updating database: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    DATABASE_TOOLS.delete_database,
    "Delete a database connection",
    {
      database_id: z.string().describe("The ID of the database to delete"),
      confirmation: z.string().describe('Required: DELETE:<database_id>'),
    },
    async ({ database_id, confirmation }) => {
      try {
        requireDestructiveConfirmation(confirmation, 'DELETE', database_id);
        await client.deleteDatabase(database_id);
        return {
          content: [
            {
              type: "text",
              text: `Database ${database_id} deleted successfully.`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error deleting database: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );

  server.tool(
    DATABASE_TOOLS.test_database_connection,
    "Test a database connection",
    {
      database_id: z.string().describe("The ID of the database to test"),
    },
    async ({ database_id }) => {
      try {
        const response = await client.testDatabaseConnection(database_id);
        return {
          content: [
            {
              type: "text",
              text: `Database connection test result: ${JSON.stringify(response, null, 2)}`,
            },
          ],
        };
      } catch (error) {
        return {
          content: [
            {
              type: "text",
              text: `Error testing database connection: ${error instanceof Error ? error.message : String(error)}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

export { DATABASE_TOOLS };
