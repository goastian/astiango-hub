import { chmodSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { loadMcpConfiguration, startupStatusMessages } from './config.js';

describe('MCP configuration', () => {
  let directory: string;
  let tokenFile: string;

  beforeEach(() => {
    directory = mkdtempSync(join(tmpdir(), 'astiango-mcp-'));
    tokenFile = join(directory, 'api-token');
    writeFileSync(tokenFile, 'unit-test-token\n', { mode: 0o600 });
    chmodSync(tokenFile, 0o600);
  });

  afterEach(() => rmSync(directory, { recursive: true, force: true }));

  it('loads a token only from a private secret file', () => {
    expect(
      loadMcpConfiguration(['https://hub.example.test/api'], {
        ASTIANGO_API_TOKEN_FILE: tokenFile,
      })
    ).toEqual({
      apiEndpoint: 'https://hub.example.test/api',
      apiToken: 'unit-test-token',
      allowMutations: false,
    });
  });

  it('rejects tokens passed as command arguments', () => {
    expect(() =>
      loadMcpConfiguration(['https://hub.example.test/api', 'never-accept-cli-token'], {
        ASTIANGO_API_TOKEN_FILE: tokenFile,
      })
    ).toThrow('never as a command argument');
  });

  it('rejects a missing secret file configuration', () => {
    expect(() => loadMcpConfiguration(['https://hub.example.test/api'], {})).toThrow(
      'ASTIANGO_API_TOKEN_FILE is required'
    );
  });

  it('rejects insecure secret permissions on POSIX', () => {
    if (process.platform === 'win32') {
      return;
    }
    chmodSync(tokenFile, 0o644);
    expect(() =>
      loadMcpConfiguration(['https://hub.example.test/api'], { ASTIANGO_API_TOKEN_FILE: tokenFile })
    ).toThrow('must not be readable by group or others');
  });

  it('does not include the token in startup status messages', () => {
    const configuration = loadMcpConfiguration(['https://hub.example.test/api'], {
      ASTIANGO_API_TOKEN_FILE: tokenFile,
    });
    expect(startupStatusMessages(configuration).join('\n')).not.toContain('unit-test-token');
  });
});
