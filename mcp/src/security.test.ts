import { isReadOnlyMethod, mutationsAreEnabled, requireDestructiveConfirmation } from './security.js';

describe('MCP security controls', () => {
  it('keeps mutations disabled unless explicitly enabled', () => {
    expect(mutationsAreEnabled(undefined)).toBe(false);
    expect(mutationsAreEnabled('false')).toBe(false);
    expect(mutationsAreEnabled('true')).toBe(true);
  });

  it('only treats safe HTTP methods as read-only', () => {
    expect(isReadOnlyMethod('get')).toBe(true);
    expect(isReadOnlyMethod('HEAD')).toBe(true);
    expect(isReadOnlyMethod('post')).toBe(false);
    expect(isReadOnlyMethod('delete')).toBe(false);
  });

  it('requires an operation-specific destructive confirmation', () => {
    expect(() => requireDestructiveConfirmation('DELETE:spider-1', 'DELETE', 'spider-1')).not.toThrow();
    expect(() => requireDestructiveConfirmation('DELETE:other', 'DELETE', 'spider-1')).toThrow(
      'Confirmation required'
    );
  });
});
