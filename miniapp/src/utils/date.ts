/**
 * 轻量日期与时间处理工具
 * 零依赖，替代 moment.js / dayjs，节省几十到数百 KB 体积
 */

/**
 * 安全解析日期对象（兼容 iOS 下对 '2026-09-09' 的解析兼容性）
 */
function parseDate(input: string | number | Date): Date | null {
  if (!input) return null
  if (input instanceof Date) return isNaN(input.getTime()) ? null : input

  if (typeof input === 'string') {
    // 兼容 iOS 将 "YYYY-MM-DD" 替换为 "YYYY/MM/DD"
    const normalized = input.includes('-') && !input.includes('T') ? input.replace(/-/g, '/') : input
    const d = new Date(normalized)
    return isNaN(d.getTime()) ? null : d
  }

  if (typeof input === 'number') {
    const timestamp = input.toString().length === 10 ? input * 1000 : input
    const d = new Date(timestamp)
    return isNaN(d.getTime()) ? null : d
  }

  return null
}

/**
 * 格式化日期时间
 * 支持模板标识: YYYY (年), MM (月), DD (日), HH (24时), mm (分), ss (秒)
 * 默认格式: 'YYYY-MM-DD HH:mm:ss'
 */
export function formatDate(
  input?: string | number | Date | null,
  format = 'YYYY-MM-DD HH:mm:ss'
): string {
  if (!input) return ''
  const date = parseDate(input)
  if (!date) return ''

  const year = date.getFullYear()
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  const seconds = date.getSeconds().toString().padStart(2, '0')

  return format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

/**
 * 订单/活动倒计时格式化
 * @param remainingSeconds 剩余秒数
 * @returns 拆解后的天、时、分、秒及结束标志
 */
export function formatCountdown(remainingSeconds: number): {
  days: number
  hours: string
  minutes: string
  seconds: string
  isEnded: boolean
} {
  const total = Math.max(0, Math.floor(remainingSeconds))
  if (total <= 0) {
    return { days: 0, hours: '00', minutes: '00', seconds: '00', isEnded: true }
  }

  const days = Math.floor(total / (24 * 3600))
  const hours = Math.floor((total % (24 * 3600)) / 3600)
    .toString()
    .padStart(2, '0')
  const minutes = Math.floor((total % 3600) / 60)
    .toString()
    .padStart(2, '0')
  const seconds = Math.floor(total % 60)
    .toString()
    .padStart(2, '0')

  return {
    days,
    hours,
    minutes,
    seconds,
    isEnded: false,
  }
}
