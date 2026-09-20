import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  wechatLogin,
  login,
  register,
  getMemberMe,
  updateMemberProfile,
  uploadMemberAvatar,
  getMemberAddresses,
  createMemberAddress,
  updateMemberAddress,
  deleteMemberAddress,
  setDefaultMemberAddress,
  type MemberInfo,
  type AddressResponse,
  type AddressCreateRequest,
  type AddressUpdateRequest,
  type LoginRequest,
  type RegisterRequest,
} from '@/api'
import {
  getToken,
  setToken,
  getStoredMember,
  setStoredMember,
  clearAuthStorage,
  resolveImageUrl,
} from '@/utils'
import { useCartStore } from './cart'

/**
 * 规范化会员头像地址
 */
function normalizeMember(info: MemberInfo): MemberInfo {
  if (info && info.avatar_url) {
    info.avatar_url = resolveImageUrl(info.avatar_url)
  }
  return info
}

export const useMemberStore = defineStore('member', () => {
  // 会员认证与资料状态初始化：从统一 auth-storage 读取持久化缓存
  const token = ref<string>(getToken())
  const cachedMember = getStoredMember<MemberInfo>()
  const member = ref<MemberInfo | null>(cachedMember ? normalizeMember(cachedMember) : null)
  const addresses = ref<AddressResponse[]>([])
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  // 严格定义：Token 与会员资料同时就绪才视为已登录
  const isLoggedIn = computed<boolean>(() => {
    return !!token.value && !!member.value
  })

  /**
   * 凭微信 Code 登录并同步会员资料
   */
  async function loginWithCode(code: string) {
    loading.value = true
    error.value = null
    try {
      const res = await wechatLogin(code)
      if (res && res.code === 1 && res.data) {
        const { token: jwtToken, member: memberData } = res.data
        token.value = jwtToken
        member.value = normalizeMember(memberData)

        // 统一持久化存储
        setToken(jwtToken)
        setStoredMember(member.value)

        // 异步静默拉取地址列表与购物车
        fetchAddresses().catch(() => {})
        try {
          useCartStore().fetchCart().catch(() => {})
        } catch {}

        return res.data
      }
      throw new Error(res?.message || '登录失败')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '登录失败'
      error.value = msg
      console.warn('会员登录失败:', msg)
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * 手机号密码登录 (POST /api/miniapp/auth/login)
   */
  async function loginWithPassword(req: LoginRequest) {
    loading.value = true
    error.value = null
    try {
      const res = await login(req)
      if (res && res.code === 1 && res.data) {
        const { token: jwtToken, member: memberData } = res.data
        token.value = jwtToken
        member.value = normalizeMember(memberData)

        setToken(jwtToken)
        setStoredMember(member.value)

        fetchAddresses().catch(() => {})
        try {
          useCartStore().fetchCart().catch(() => {})
        } catch {}

        return res.data
      }
      throw new Error(res?.message || '登录失败')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '登录失败'
      error.value = msg
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * 手机号密码注册 (POST /api/miniapp/auth/register)
   */
  async function registerWithPassword(req: RegisterRequest) {
    loading.value = true
    error.value = null
    try {
      const res = await register(req)
      if (res && res.code === 1 && res.data) {
        const { token: jwtToken, member: memberData } = res.data
        token.value = jwtToken
        member.value = normalizeMember(memberData)

        setToken(jwtToken)
        setStoredMember(member.value)

        fetchAddresses().catch(() => {})
        try {
          useCartStore().fetchCart().catch(() => {})
        } catch {}

        return res.data
      }
      throw new Error(res?.message || '注册失败')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '注册失败'
      error.value = msg
      throw err
    } finally {
      loading.value = false
    }
  }

  // 异步请求序号追踪，防止乱序与跨账号响应覆盖
  let refreshSeq = 0
  let addressSeq = 0

  /**
   * 刷新当前会员资料 (GET /api/miniapp/member/me)
   * 启动流程或网络恢复时用于同步最新资料，失败保留本地 member 缓存
   */
  async function refreshMember(): Promise<MemberInfo | null> {
    const requestToken = getToken()
    if (!requestToken) return null
    const seq = ++refreshSeq
    try {
      const res = await getMemberMe()
      // 跨账号或乱序请求防护：若 token 改变或不是最新请求序号，丢弃结果禁止写入 Store
      if (requestToken !== getToken() || seq !== refreshSeq) {
        return member.value
      }
      if (res && res.code === 1 && res.data) {
        member.value = normalizeMember(res.data)
        setStoredMember(member.value)
        return member.value
      }
      return member.value
    } catch (err) {
      if (requestToken !== getToken() || seq !== refreshSeq) {
        return member.value
      }
      console.warn('[MemberStore] 获取会员资料失败:', err)
      return member.value
    }
  }

  /**
   * 更新会员昵称 (POST /api/miniapp/member/profile)
   */
  async function updateNickname(nickname: string): Promise<MemberInfo | null> {
    const requestToken = getToken()
    if (!requestToken) return null
    loading.value = true
    try {
      const res = await updateMemberProfile({ nickname })
      if (requestToken !== getToken()) {
        return member.value
      }
      if (res && res.code === 1 && res.data) {
        member.value = normalizeMember(res.data)
        setStoredMember(member.value)
        return member.value
      }
      return member.value
    } finally {
      if (requestToken === getToken()) {
        loading.value = false
      }
    }
  }

  /**
   * 上传并更新会员头像 (POST /api/miniapp/member/avatar)
   */
  async function updateAvatar(filePath: string) {
    const requestToken = getToken()
    if (!requestToken) return null
    loading.value = true
    try {
      const res = await uploadMemberAvatar(filePath)
      if (requestToken !== getToken()) {
        return null
      }
      if (res && res.code === 1 && res.data) {
        if (member.value) {
          member.value.avatar_id = res.data.avatar_id
          member.value.avatar_url = resolveImageUrl(res.data.avatar_url)
          setStoredMember(member.value)
        }
        return res.data
      }
      return null
    } finally {
      if (requestToken === getToken()) {
        loading.value = false
      }
    }
  }

  /**
   * 获取收货地址列表 (GET /api/miniapp/member/addresses)
   */
  async function fetchAddresses(): Promise<AddressResponse[]> {
    const requestToken = getToken()
    if (!requestToken) {
      addresses.value = []
      return []
    }
    const seq = ++addressSeq
    try {
      const res = await getMemberAddresses()
      if (requestToken !== getToken() || seq !== addressSeq) {
        return addresses.value
      }
      if (res && res.code === 1 && res.data) {
        addresses.value = res.data
        return res.data
      }
      return addresses.value
    } catch (err) {
      if (requestToken !== getToken() || seq !== addressSeq) {
        return addresses.value
      }
      console.warn('[MemberStore] 拉取地址列表失败:', err)
      return addresses.value
    }
  }

  /**
   * 新增收货地址 (POST /api/miniapp/member/addresses/create)
   */
  async function addAddress(req: AddressCreateRequest) {
    const res = await createMemberAddress(req)
    await fetchAddresses()
    return res
  }

  /**
   * 更新收货地址 (POST /api/miniapp/member/addresses/update)
   */
  async function editAddress(req: AddressUpdateRequest) {
    const res = await updateMemberAddress(req)
    await fetchAddresses()
    return res
  }

  /**
   * 删除收货地址 (POST /api/miniapp/member/addresses/delete)
   */
  async function removeAddress(id: number) {
    const res = await deleteMemberAddress(id)
    await fetchAddresses()
    return res
  }

  /**
   * 设为默认收货地址 (POST /api/miniapp/member/addresses/default)
   */
  async function makeDefaultAddress(id: number) {
    const res = await setDefaultMemberAddress(id)
    await fetchAddresses()
    return res
  }

  /**
   * 退出登录：统一清理 Store 内存与本地 Storage，清理结算临时态，并重置关联 Store (如购物车)
   */
  function logout() {
    refreshSeq++
    addressSeq++
    token.value = ''
    member.value = null
    addresses.value = []
    loading.value = false
    error.value = null
    clearAuthStorage()
    uni.removeStorageSync('checkout_payload')

    // 跨 Store 协同清理：重置购物车 Store，避免不同账号数据污染
    try {
      const cartStore = useCartStore()
      cartStore.reset()
    } catch (err) {
      console.warn('[MemberStore] logout reset cart failed:', err)
    }
  }

  return {
    token,
    member,
    addresses,
    loading,
    error,
    isLoggedIn,
    loginWithCode,
    loginWithPassword,
    registerWithPassword,
    refreshMember,
    updateNickname,
    updateAvatar,
    fetchAddresses,
    addAddress,
    editAddress,
    removeAddress,
    makeDefaultAddress,
    logout,
  }
})
