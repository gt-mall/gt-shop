export const MAX_SAFE_PRODUCT_ID = 9007199254740991 // Number.MAX_SAFE_INTEGER
export const MAX_SCENE_LENGTH = 128

export interface ParseProductEntryResult {
  valid: boolean
  productId?: number
  error?: string
}

export interface ProductShareOptions {
  product?: {
    id: number
    name: string
    main_image_url?: string
  } | null
  shopName?: string
  defaultImageUrl?: string
}

export interface ProductShareData {
  title: string
  path: string
  query: string
  imageUrl: string
}

export interface LocationLike {
  origin: string
  pathname: string
  hash: string
  href?: string
}

/**
 * 校验并解析正整数字符串/数字
 */
export function parsePositiveInteger(val: unknown): number | null {
  if (typeof val === 'number') {
    if (Number.isInteger(val) && val > 0 && val <= MAX_SAFE_PRODUCT_ID) {
      return val
    }
    return null
  }
  if (typeof val === 'string') {
    const trimmed = val.trim()
    // 严格匹配正整数，不含前导零（除单个数字外）或非法符号
    if (!/^[1-9]\d{0,15}$/.test(trimmed)) {
      return null
    }
    const num = Number(trimmed)
    if (Number.isSafeInteger(num) && num > 0) {
      return num
    }
  }
  return null
}

/**
 * 解析并校验进入商品详情页的参数：
 * 1. 支持普通 query.id
 * 2. 支持小程序码扫码参数 query.scene（约定短场景：p=<商品ID>）
 * 3. 校验正整数、长度及安全范围
 * 4. 出现两者时必须一致，冲突或非法时安全报错
 * 5. 防御微信启动场景值（数字）与扫码 scene 混淆
 */
export function parseProductEntry(query?: Record<string, unknown> | null): ParseProductEntryResult {
  if (!query || typeof query !== 'object') {
    return { valid: false, error: '缺少商品ID参数' }
  }

  let idFromQuery: number | null = null
  let hasIdField = false

  if (query.id !== undefined && query.id !== null && query.id !== '') {
    hasIdField = true
    idFromQuery = parsePositiveInteger(query.id)
    if (idFromQuery === null) {
      return { valid: false, error: '商品ID参数非法' }
    }
  }

  let idFromScene: number | null = null
  let hasSceneField = false

  if (query.scene !== undefined && query.scene !== null && query.scene !== '') {
    // 只有字符串类型的 scene 才是小程序码传递的自定义场景参数
    // 数值类型（如 1001）为微信启动入口场景值，不作为商品自定义 scene 处理
    if (typeof query.scene === 'string') {
      const trimmed = query.scene.trim()
      if (trimmed) {
        hasSceneField = true
        if (trimmed.length > MAX_SCENE_LENGTH) {
          return { valid: false, error: '场景参数超出长度限制' }
        }

        let decoded = ''
        try {
          decoded = decodeURIComponent(trimmed)
        } catch {
          return { valid: false, error: '场景参数解码失败' }
        }

        // 解析约定格式 p=<id> 或 id=<id>
        const pairs = decoded.split('&')
        let foundProductKey = false
        for (const pair of pairs) {
          const eqIdx = pair.indexOf('=')
          if (eqIdx > 0) {
            const key = pair.slice(0, eqIdx).trim()
            const val = pair.slice(eqIdx + 1).trim()
            if (key === 'p' || key === 'id') {
              foundProductKey = true
              const parsed = parsePositiveInteger(val)
              if (parsed === null) {
                return { valid: false, error: '场景中的商品ID非法' }
              }
              idFromScene = parsed
              break
            }
          }
        }
        if (!foundProductKey) {
          return { valid: false, error: '场景中未找到有效商品标识' }
        }
      }
    }
  }

  // 两者都存在时必须完全一致
  if (hasIdField && hasSceneField) {
    if (idFromQuery !== null && idFromScene !== null) {
      if (idFromQuery !== idFromScene) {
        return { valid: false, error: '商品参数冲突' }
      }
      return { valid: true, productId: idFromQuery }
    }
  }

  if (hasIdField && idFromQuery !== null) {
    return { valid: true, productId: idFromQuery }
  }

  if (hasSceneField && idFromScene !== null) {
    return { valid: true, productId: idFromScene }
  }

  return { valid: false, error: '缺少商品ID参数' }
}

/**
 * 组装原生微信商品分享数据
 * 规范：未加载、下架或无效商品不能伪造可购买分享（返回 null）
 */
export function buildProductShare(options: ProductShareOptions): ProductShareData | null {
  const { product, shopName, defaultImageUrl = '' } = options
  if (!product || !product.id || typeof product.id !== 'number' || product.id <= 0) {
    return null
  }

  const cleanTitle = (product.name || '').trim() || shopName || '精选好物'
  const path = `/pages/product/detail?id=${product.id}`
  const query = `id=${product.id}`
  const imageUrl = (product.main_image_url || '').trim() || defaultImageUrl

  return {
    title: cleanTitle,
    path,
    query,
    imageUrl,
  }
}

/**
 * 组装 H5 环境下的商品复制链接
 * 动态读取当前站点的 origin、pathname 与 hash 模式，严禁硬编码开发环境或未知域名
 */
export function buildH5ShareUrl(productId: number, loc?: LocationLike | null): string {
  const safeId = parsePositiveInteger(productId)
  if (!safeId) {
    return ''
  }

  const currentLoc: LocationLike | null = loc || (typeof window !== 'undefined' ? window.location : null)
  if (!currentLoc) {
    return `/pages/product/detail?id=${safeId}`
  }

  const origin = currentLoc.origin || ''
  const pathname = currentLoc.pathname || ''
  const hash = currentLoc.hash || ''

  // Hash 路由模式
  if (hash && hash.startsWith('#/')) {
    return `${origin}${pathname}#/pages/product/detail?id=${safeId}`
  }

  // History 路由模式
  const base = pathname.replace(/\/pages\/.*$/, '').replace(/\/+$/, '')
  return `${origin}${base}/pages/product/detail?id=${safeId}`
}
