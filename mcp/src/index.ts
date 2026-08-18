#!/usr/bin/env node

import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { configurePrompts } from './prompts.js';
import { configureAllTools } from './tools.js';
import { AstianGoHubClient } from './client.js';
import { loadMcpConfiguration, startupStatusMessages } from './config.js';
import { packageVersion } from './version.js';

async function main() {
  const configuration = loadMcpConfiguration(process.argv.slice(2));
  const server = new McpServer({
    name: 'AstianGO Hub MCP Server',
    version: packageVersion,
  });

  // Initialize AstianGO Hub client
  const client = new AstianGoHubClient(
    configuration.apiEndpoint,
    configuration.apiToken,
    30000,
    configuration.allowMutations
  );

  // Configure prompts and tools
  configurePrompts(server);
  configureAllTools(server, client);

  const transport = new StdioServerTransport();
  console.error(`AstianGO Hub MCP Server version: ${packageVersion}`);
  for (const message of startupStatusMessages(configuration)) {
    console.error(message);
  }

  await server.connect(transport);
}

main().catch(error => {
  console.error('Fatal error in main():', error);
  process.exit(1);
});
