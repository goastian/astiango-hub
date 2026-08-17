import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { AstianGoHubClient } from "./client.js";

import { configureSpiderTools } from './tools/spiders.js';
import { configureTaskTools } from './tools/tasks.js';
import { configureNodeTools } from './tools/nodes.js';
import { configureScheduleTools } from './tools/schedules.js';
import { configureSystemTools } from './tools/system.js';
import { configureProjectTools } from './tools/projects.js';
import { configureDatabaseTools } from './tools/databases.js';
import { configureGitTools } from './tools/git.js';
import { configureStatsTools } from './tools/stats.js';
import { configureAITools } from './tools/ai.js';

export function configureAllTools(server: McpServer, client: AstianGoHubClient) {
  configureSpiderTools(server, client);
  configureTaskTools(server, client);
  configureNodeTools(server, client);
  configureScheduleTools(server, client);
  configureSystemTools(server, client);
  configureProjectTools(server, client);
  configureDatabaseTools(server, client);
  configureGitTools(server, client);
  configureStatsTools(server, client);
  configureAITools(server, client);
}
