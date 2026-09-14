/**
 * 会话 Token 一致性判断（纯函数，无任何运行时依赖，便于直接单元测试）。
 *
 * 只有“触发认证失效的请求所使用的 Token”仍然等于“当前 Storage 中的 Token”时，
 * 才允许执行全局登出；若 Token 已变化，说明用户已经 logout、切换账号或重新登录，
 * 旧请求的认证失败必须被忽略，不能影响当前账号。
 */
export function isCurrentSessionToken(
  requestToken: string,
  currentToken: string,
): boolean {
  return !!requestToken && requestToken === currentToken;
}
