import { readFileSync, statSync } from 'node:fs';

import { mutationsAreEnabled } from './security.js';

export interface McpConfiguration {
  apiEndpoint: string;
  apiToken: string;
  allowMutations: boolean;
}

type Environment = NodeJS.ProcessEnv;

function readSecretFile(secretFile: string): string {
  const stat = statSync(secretFile);
  if (!stat.isFile()) {
    throw new Error('ASTIANGO_API_TOKEN_FILE must point to a regular file.');
  }

  // Docker/Kubernetes secrets should be private to their owner. Windows does
  // not expose POSIX mode bits, so the check is meaningful only on POSIX.
  if (process.platform !== 'win32' && (stat.mode & 0o077) !== 0) {
    throw new Error('ASTIANGO_API_TOKEN_FILE must not be readable by group or others.');
  }

  const token = readFileSync(secretFile, 'utf8').trim();
  if (!token) {
    throw new Error('ASTIANGO_API_TOKEN_FILE is empty.');
  }
  return token;
}

export function loadMcpConfiguration(
  args: string[],
  environment: Environment = process.env
): McpConfiguration {
  if (args.length > 1) {
    throw new Error('API tokens must be supplied through ASTIANGO_API_TOKEN_FILE, never as a command argument.');
  }

  const apiEndpoint = args[0] ?? environment.ASTIANGO_API_URL ?? environment.ASTIANGO_API_ENDPOINT;
  if (!apiEndpoint) {
    throw new Error('AstianGO Hub API URL is required. Use one command argument or ASTIANGO_API_ENDPOINT.');
  }

  const secretFile = environment.ASTIANGO_API_TOKEN_FILE;
  if (!secretFile) {
    throw new Error('ASTIANGO_API_TOKEN_FILE is required. Mount the API token as a private secret file.');
  }

  return {
    apiEndpoint,
    apiToken: readSecretFile(secretFile),
    allowMutations: mutationsAreEnabled(environment.ASTIANGO_MCP_ALLOW_MUTATIONS),
  };
}

export function startupStatusMessages(configuration: McpConfiguration): string[] {
  return [
    `AstianGO Hub API endpoint: ${configuration.apiEndpoint}`,
    `Mutating MCP operations: ${configuration.allowMutations ? 'enabled' : 'disabled (default)'}`,
  ];
}
