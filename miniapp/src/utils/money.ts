/**
 * 商城金额计算与格式化工具
 * 遵循规范：金额计算精确转换为整数分，严禁直接使用 JS 浮点数加减乘除导致失真
 */

/**
 * 将元（字符串或数字）安全转换为整数分
 * 采用字符串精确解析，避免 2.09 * 100 = 208.99999999999997 浮点精度陷阱
 */
export function yuanToFen(yuan: string | number): number {
  if (yuan === undefined || yuan === null || yuan === '') return 0
  const str = String(yuan).trim()
  if (!str || isNaN(Number(str))) return 0

  const isNegative = str.startsWith('-')
  const absStr = isNegative ? str.slice(1) : str
  const [intPart = '0', fracPart = ''] = absStr.split('.')

  // 截取并补齐前两位小数
  const paddedFrac = (fracPart + '00').slice(0, 2)
  const extraFrac = fracPart.slice(2)

  let fen = parseInt(intPart, 10) * 100 + parseInt(paddedFrac, 10)
  // 如果存在第三位小数，四舍五入
  if (extraFrac && parseInt(extraFrac[0], 10) >= 5) {
    fen += 1
  }

  return isNegative ? -fen : fen
}

/**
 * 将整数分安全转换为标准两位小数元字符串
 * 例如: 209 -> '2.09', 50 -> '0.50'
 */
export function fenToYuan(fen: number): string {
  if (!fen || isNaN(fen)) return '0.00'
  const isNegative = fen < 0
  const absFen = Math.abs(Math.round(fen))
  const intPart = Math.floor(absFen / 100).toString()
  const fracPart = (absFen % 100).toString().padStart(2, '0')
  return `${isNegative ? '-' : ''}${intPart}.${fracPart}`
}

/**
 * 安全金额加法（输入元，返回元）
 */
export function safeAdd(a: string | number, b: string | number): string {
  const fenA = yuanToFen(a)
  const fenB = yuanToFen(b)
  return fenToYuan(fenA + fenB)
}

/**
 * 安全金额减法（输入元，返回元）
 */
export function safeSub(a: string | number, b: string | number): string {
  const fenA = yuanToFen(a)
  const fenB = yuanToFen(b)
  return fenToYuan(fenA - fenB)
}

/**
 * 安全金额乘法（输入元和数量，返回元）
 * @param yuan 单价（元）
 * @param quantity 购买件数
 */
export function safeMul(yuan: string | number, quantity: number): string {
  const fen = yuanToFen(yuan)
  const totalFen = Math.round(fen * quantity)
  return fenToYuan(totalFen)
}

/**
 * 金额展示格式化
 * 例如: 12.3 -> '¥12.30'
 */
export function formatMoney(
  amount: string | number,
  options: { prefix?: string; fallback?: string } = {}
): string {
  const { prefix = '¥', fallback = '¥0.00' } = options
  if (amount === undefined || amount === null || amount === '') return fallback

  const fen = yuanToFen(amount)
  return `${prefix}${fenToYuan(fen)}`
}
