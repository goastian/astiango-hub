const SAFE_HTTP_METHODS = new Set(['get', 'head', 'options']);

export function mutationsAreEnabled(value: string | undefined): boolean {
  return value === 'true';
}

export function isReadOnlyMethod(method: string | undefined): boolean {
  return SAFE_HTTP_METHODS.has((method ?? 'get').toLowerCase());
}

export function requireDestructiveConfirmation(
  confirmation: string,
  action: string,
  resourceID: string
): void {
  const expected = `${action}:${resourceID}`;
  if (confirmation !== expected) {
    throw new Error(`Confirmation required. Set confirmation to "${expected}".`);
  }
}
