/**
 * 微信收货地址导入 Composable
 * 支持地址列表页、编辑页、结算页复用
 * 严格遵从登录鉴权、时序锁保护、敏感信息内存暂存、真实区划树匹配与单客户端去重防重
 */
import { ref } from "vue"
import { storeToRefs } from "pinia"
import { useMemberStore } from "@/stores/member"
import { getRegions, type RegionItem } from "@/api"
import {
  matchRegionFromTree,
  getCachedRegionTree,
  setCachedRegionTree,
  findMatchingAddress,
} from "@/utils/addressParse"

export interface WechatAddressDraft {
  userId: number | string
  name: string
  mobile: string
  province: string
  province_code: string
  city: string
  city_code: string
  district: string
  district_code: string
  address: string
  postal_code: string
  regionMatched: boolean
  unmatchedReason?: string
}

// 内存暂存草稿，仅在本次会话内通过组件/路由流转，不写入持久化 Storage，离开页面后清理
let sessionDraft: WechatAddressDraft | null = null

export function setWechatAddressDraft(draft: WechatAddressDraft): void {
  sessionDraft = draft
}

export function consumeWechatAddressDraft(currentUserId: number | string): WechatAddressDraft | null {
  if (!sessionDraft) return null
  // 跨账号校验：若账号已变更，直接作废销毁
  if (sessionDraft.userId !== currentUserId) {
    sessionDraft = null
    return null
  }
  const draft = sessionDraft
  sessionDraft = null
  return draft
}

export function clearWechatAddressDraft(): void {
  sessionDraft = null
}

export interface UseWechatAddressImportOptions {
  fromSelect?: boolean | (() => boolean)
  onDraft?: (draft: WechatAddressDraft) => void
}

export function useWechatAddressImport(options?: UseWechatAddressImportOptions) {
  const memberStore = useMemberStore()
  const { token, addresses, member } = storeToRefs(memberStore)

  // 全局/局部操作防重锁
  const isImporting = ref<boolean>(false)

  function getIsFromSelect(): boolean {
    if (typeof options?.fromSelect === "function") {
      return options.fromSelect()
    }
    return !!options?.fromSelect
  }

  async function importWechatAddress(): Promise<void> {
    if (!token.value) {
      uni.showToast({ title: "请先登录后导入收货地址", icon: "none" })
      setTimeout(() => {
        uni.switchTab({ url: "/pages/my/index" })
      }, 800)
      return
    }

    // #ifndef MP-WEIXIN
    uni.showToast({ title: "当前环境不支持微信地址导入", icon: "none" })
    return
    // #endif

    // #ifdef MP-WEIXIN
    if (isImporting.value) return
    isImporting.value = true

    const requestToken = token.value
    const requestUserId = member.value?.id || 0

    // 确保已获取全国区划树
    let tree: RegionItem[] = getCachedRegionTree() || []
    if (tree.length === 0) {
      try {
        const res = await getRegions()
        if (res && res.code === 1 && res.data?.regions) {
          tree = res.data.regions
          setCachedRegionTree(tree)
        }
      } catch {
        // 网络请求失败将在匹配时进行兜底提示，不阻断流程
      }
    }

    uni.chooseAddress({
      success: async (res) => {
        // 1. 账号变动安全检查：若原生弹窗期间登录态发生变化，立即丢弃
        if (token.value !== requestToken) {
          isImporting.value = false
          return
        }

        if (!res.userName || !res.telNumber) {
          uni.showToast({ title: "微信地址信息不完整", icon: "none" })
          isImporting.value = false
          return
        }

        const rawName = (res.userName || "").trim()
        const rawMobile = (res.telNumber || "").trim()
        const rawProvince = (res.provinceName || "").trim()
        const rawCity = (res.cityName || "").trim()
        const rawDistrict = (res.countyName || "").trim()
        const rawStreet = (res.streetName || "").trim()
        const rawDetail = (res.detailInfo || "").trim()
        const nationalCode = (res.nationalCode || "").trim()
        const postalCode = (res.postalCode || "").trim()

        // 2. 街道与详细地址处理：避免重复拼接
        let finalAddress = rawDetail
        if (rawStreet) {
          if (!finalAddress.startsWith(rawStreet)) {
            finalAddress = `${rawStreet}${finalAddress}`
          }
        }

        // 3. 基于真实区划树精准匹配三级代码（禁止猜测编码）
        const matchResult = tree
          ? matchRegionFromTree(tree, rawProvince, rawCity, rawDistrict, nationalCode)
          : {
              matched: false,
              province: rawProvince,
              province_code: "",
              city: rawCity,
              city_code: "",
              district: rawDistrict,
              district_code: "",
              unmatchedReason: "区划树未就绪",
            }

        // 4. 单客户端防重校验：检查当前账号是否已存在完全一致的地址
        if (matchResult.matched) {
          const existing = findMatchingAddress(addresses.value, {
            name: rawName,
            mobile: rawMobile,
            province_code: matchResult.province_code,
            city_code: matchResult.city_code,
            district_code: matchResult.district_code,
            address: finalAddress,
          })

          if (existing) {
            isImporting.value = false
            const isFromSelect = getIsFromSelect()
            if (isFromSelect) {
              uni.showToast({ title: "已复用已有相同地址", icon: "success" })
              uni.$emit("selectAddress", existing)
              setTimeout(() => {
                uni.navigateBack()
              }, 600)
            } else {
              uni.showToast({ title: "该收货地址已存在", icon: "none" })
            }
            return
          }
        }

        const draft: WechatAddressDraft = {
          userId: requestUserId,
          name: rawName,
          mobile: rawMobile,
          province: matchResult.province || rawProvince,
          province_code: matchResult.province_code,
          city: matchResult.city || rawCity,
          city_code: matchResult.city_code,
          district: matchResult.district || rawDistrict,
          district_code: matchResult.district_code,
          address: finalAddress,
          postal_code: postalCode,
          regionMatched: matchResult.matched,
          unmatchedReason: matchResult.unmatchedReason,
        }

        isImporting.value = false

        // 若调用方提供了 onDraft 回调（例如已在 edit.vue 页面中），直接就地填充
        if (options?.onDraft) {
          options.onDraft(draft)
          return
        }

        // 5. 暂存草稿，导航至编辑表单供用户最终确认
        setWechatAddressDraft(draft)

        const fromSelectParam = getIsFromSelect() ? "1" : "0"
        uni.navigateTo({
          url: `/pages/address/edit?fromWechat=1&fromSelect=${fromSelectParam}`,
        })
      },
      fail: (err: any) => {
        isImporting.value = false
        const errMsg = err?.errMsg || ""

        // 主动取消，静默结束
        if (errMsg.includes("cancel")) {
          return
        }

        // 权限拒绝
        if (
          errMsg.includes("auth deny") ||
          errMsg.includes("authorize:fail") ||
          errMsg.includes("auth denied")
        ) {
          uni.showToast({
            title: "未授权通讯地址权限，可手动输入收货地址",
            icon: "none",
            duration: 2500,
          })
          return
        }

        // 隐私协议未同意
        if (errMsg.includes("requirePrivacyContract") || errMsg.includes("privacy")) {
          uni.showToast({
            title: "请先同意用户隐私指引后再使用该功能",
            icon: "none",
            duration: 2500,
          })
          return
        }

        // 其他失败
        uni.showToast({
          title: "未能获取微信收货地址，请重试或手动输入",
          icon: "none",
          duration: 2500,
        })
      },
    })
    // #endif
  }

  return {
    isImporting,
    importWechatAddress,
  }
}
