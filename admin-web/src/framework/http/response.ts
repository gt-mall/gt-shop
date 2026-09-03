export function isSuccess(result: { code?: number } | null | undefined): boolean {
  return typeof result?.code === 'number' && result.code > 0;
}

export function isSessionExpired(result: { code?: number } | null | undefined): boolean {
  return result?.code === -1;
}

export function getMessage(result: { message?: string } | null | undefined): string {
  return result?.message ?? '请求失败';
}
