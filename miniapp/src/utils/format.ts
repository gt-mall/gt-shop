/**
 * 业务数据脱敏与文本格式化工具
 */

/**
 * 手机号脱敏
 * 例如: '13812345678' -> '138****5678'
 */
export function maskPhone(phone?: string | null): string {
  if (!phone) return ''
  const trimmed = String(phone).trim()
  if (trimmed.length < 7) return trimmed
  return trimmed.replace(/^(\d{3})\d+(\d{4})$/, '$1****$2')
}

/**
 * 姓名脱敏
 * 2个字: '张三' -> '张*'
 * 3个字: '李四五' -> '李*五'
 * 4个字以上: '欧阳六七' -> '欧**七'
 */
export function maskName(name?: string | null): string {
  if (!name) return ''
  const trimmed = String(name).trim()
  const len = trimmed.length
  if (len <= 1) return trimmed
  if (len === 2) return `${trimmed[0]}*`
  return `${trimmed[0]}${'*'.repeat(len - 2)}${trimmed[len - 1]}`
}

/**
 * 文本超长截断并追加省略号
 */
export function truncate(str?: string | null, maxLength = 20, suffix = '...'): string {
  if (!str) return ''
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength) + suffix
}

/**
 * 解析 URL 参数串为键值对象
 * 例如: '?id=123&from=share' -> { id: '123', from: 'share' }
 */
export function parseQuery(queryStr?: string | null): Record<string, string> {
  if (!queryStr) return {}
  const cleanStr = queryStr.includes('?') ? queryStr.split('?')[1] : queryStr
  const params: Record<string, string> = {}

  for (const pair of cleanStr.split('&')) {
    if (!pair) continue
    const [key, value = ''] = pair.split('=')
    if (key) {
      params[decodeURIComponent(key)] = decodeURIComponent(value)
    }
  }

  return params
}
