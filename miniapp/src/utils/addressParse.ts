/**
 * 智能收货地址文本解析、真实行政区划匹配与去重辅助工具
 */
import type { RegionItem, AddressResponse } from "@/api"

export interface ParsedAddress {
  name: string
  mobile: string
  province: string
  city: string
  district: string
  address: string
  province_code: string
  city_code: string
  district_code: string
  matched: boolean
}

export interface RegionMatchResult {
  matched: boolean
  province: string
  province_code: string
  city: string
  city_code: string
  district: string
  district_code: string
  unmatchedReason?: string
}

// 常见直辖市与特区
const MUNICIPALITIES = ["北京市", "上海市", "天津市", "重庆市"]

// 全局内存缓存区划树
let globalCachedRegionTree: RegionItem[] | null = null

export function setCachedRegionTree(tree: RegionItem[]): void {
  if (Array.isArray(tree) && tree.length > 0) {
    globalCachedRegionTree = tree
  }
}

export function getCachedRegionTree(): RegionItem[] | null {
  return globalCachedRegionTree
}

/**
 * 净化行政区划名称，剥离常见层级后缀便于模糊比对
 */
export function cleanRegionName(name: string): string {
  if (!name) return ""
  let s = name.trim()
  const suffixes = [
    "维吾尔自治区", "回族自治区", "壮族自治区", "自治区",
    "特别行政区", "林区", "矿区", "省", "市", "地区", "自治州",
    "自治县", "自治旗", "联合旗", "旗", "区", "县", "盟"
  ]
  for (const suffix of suffixes) {
    if (s.endsWith(suffix) && s.length > suffix.length) {
      s = s.slice(0, -suffix.length).trim()
      break
    }
  }
  return s
}

/**
 * 判断两个区划名称是否匹配
 */
export function matchRegionName(input: string, standard: string): boolean {
  const inStr = (input || "").trim()
  const stdStr = (standard || "").trim()
  if (!inStr || !stdStr) return false
  if (inStr === stdStr) return true

  const cleanIn = cleanRegionName(inStr)
  const cleanStd = cleanRegionName(stdStr)
  if (cleanIn && cleanIn === cleanStd) return true
  if (stdStr.startsWith(inStr) || (cleanStd && inStr.startsWith(cleanStd))) return true
  if (cleanIn && stdStr.startsWith(cleanIn)) return true
  return false
}

/**
 * 根据真实区划树逐级匹配省 -> 市 -> 区
 * 严格杜绝猜测编码，未匹配或歧义时返回 matched: false
 */
export function matchRegionFromTree(
  tree: RegionItem[],
  provinceName: string,
  cityName: string,
  districtName: string,
  nationalCode?: string
): RegionMatchResult {
  const failResult: RegionMatchResult = {
    matched: false,
    province: "",
    province_code: "",
    city: "",
    city_code: "",
    district: "",
    district_code: "",
  }

  if (!Array.isArray(tree) || tree.length === 0) {
    failResult.unmatchedReason = "区划数据未加载"
    return failResult
  }

  // 1. 匹配省份
  const pName = (provinceName || "").trim()
  if (!pName) {
    failResult.unmatchedReason = "缺少省份信息"
    return failResult
  }

  const matchedProvinces = tree.filter((p) => matchRegionName(pName, p.name))
  if (matchedProvinces.length === 0) {
    failResult.unmatchedReason = `未找到匹配省份: ${pName}`
    return failResult
  }
  if (matchedProvinces.length > 1) {
    failResult.unmatchedReason = `省份存在歧义: ${pName}`
    return failResult
  }

  const provinceNode = matchedProvinces[0]
  const provinceChildren = provinceNode.children || []

  // 2. 匹配城市
  let cName = (cityName || "").trim()
  let cityNode: RegionItem | null = null

  // 直辖市特殊处理：北京、天津、上海、重庆，其子节点通常只有 "市辖区"
  const isMunicipality = provinceChildren.length === 1 && provinceChildren[0].name === "市辖区"
  if (isMunicipality) {
    if (!cName || cName === "市辖区" || matchRegionName(cName, provinceNode.name)) {
      cityNode = provinceChildren[0]
    }
  }

  // 普通城市匹配
  if (!cityNode) {
    if (!cName) {
      failResult.province = provinceNode.name
      failResult.province_code = provinceNode.code
      failResult.unmatchedReason = "缺少城市信息"
      return failResult
    }

    // A. 在省份下直接匹配城市
    const matchedCities = provinceChildren.filter((c) => matchRegionName(cName, c.name))
    if (matchedCities.length === 1) {
      cityNode = matchedCities[0]
    } else if (matchedCities.length > 1) {
      failResult.province = provinceNode.name
      failResult.province_code = provinceNode.code
      failResult.unmatchedReason = `城市存在歧义: ${cName}`
      return failResult
    } else {
      // B. 可能是省直辖县级行政区划（如河南济源、湖北仙桃/天门/潜江、海南省直辖县等）
      for (const c of provinceChildren) {
        if (
          c.name.includes("直辖") ||
          c.name.includes("省直管") ||
          c.name.includes("自治区直辖")
        ) {
          const subDistricts = c.children || []
          const foundSub = subDistricts.find((d) => matchRegionName(cName, d.name))
          if (foundSub) {
            cityNode = c
            break
          }
        }
      }
    }
  }

  if (!cityNode) {
    failResult.province = provinceNode.name
    failResult.province_code = provinceNode.code
    failResult.unmatchedReason = `未找到匹配城市: ${cName}`
    return failResult
  }

  // 3. 匹配区县
  const dName = (districtName || "").trim()
  const cityChildren = cityNode.children || []
  let districtNode: RegionItem | null = null

  // 若提供了 nationalCode，在当前城市下校验确认
  if (nationalCode && nationalCode.trim()) {
    const codeMatch = cityChildren.find((d) => d.code === nationalCode.trim())
    if (codeMatch && (!dName || matchRegionName(dName, codeMatch.name))) {
      districtNode = codeMatch
    }
  }

  // 按区县名称匹配
  if (!districtNode && dName) {
    const matchedDistricts = cityChildren.filter((d) => matchRegionName(dName, d.name))
    if (matchedDistricts.length === 1) {
      districtNode = matchedDistricts[0]
    } else if (matchedDistricts.length > 1) {
      failResult.province = provinceNode.name
      failResult.province_code = provinceNode.code
      failResult.city = isMunicipality ? provinceNode.name : cityNode.name
      failResult.city_code = cityNode.code
      failResult.unmatchedReason = `区县存在歧义: ${dName}`
      return failResult
    }
  }

  // 若为省直辖县级市，微信传入的 cityName 已经是区县名（例如 cityName="济源市"），且 dName 为空或与其一致
  if (!districtNode && cityNode.name.includes("直辖")) {
    const subDistricts = cityChildren
    const foundSub = subDistricts.find(
      (d) => matchRegionName(cName, d.name) || (dName && matchRegionName(dName, d.name))
    )
    if (foundSub) {
      districtNode = foundSub
    }
  }

  if (!districtNode) {
    failResult.province = provinceNode.name
    failResult.province_code = provinceNode.code
    failResult.city = isMunicipality ? provinceNode.name : cityNode.name
    failResult.city_code = cityNode.code
    failResult.unmatchedReason = dName ? `未找到匹配区县: ${dName}` : "缺少区县信息"
    return failResult
  }

  return {
    matched: true,
    province: provinceNode.name,
    province_code: provinceNode.code,
    city: isMunicipality ? provinceNode.name : cityNode.name,
    city_code: cityNode.code,
    district: districtNode.name,
    district_code: districtNode.code,
  }
}

/**
 * 解析并获取规范行政区划代码
 * 若无法明确匹配，返回 matched: false 且区划编码置空，禁止猜测编码
 */
export function resolveRegionCodes(
  province: string,
  city?: string,
  district?: string,
  existingCodes?: { province_code?: string; city_code?: string; district_code?: string },
  tree?: RegionItem[]
): { province_code: string; city_code: string; district_code: string; matched: boolean } {
  // 1. 若已有完整的三个编码，直接沿用
  if (
    existingCodes?.province_code &&
    existingCodes?.city_code &&
    existingCodes?.district_code
  ) {
    return {
      province_code: existingCodes.province_code.trim(),
      city_code: existingCodes.city_code.trim(),
      district_code: existingCodes.district_code.trim(),
      matched: true,
    }
  }

  // 2. 若存在区划树，根据真实区划树逐级精准匹配
  const activeTree = tree || getCachedRegionTree()
  if (activeTree && activeTree.length > 0) {
    const res = matchRegionFromTree(
      activeTree,
      province,
      city || "",
      district || "",
      existingCodes?.district_code
    )
    if (res.matched) {
      return {
        province_code: res.province_code,
        city_code: res.city_code,
        district_code: res.district_code,
        matched: true,
      }
    }
    return {
      province_code: res.province_code || "",
      city_code: res.city_code || "",
      district_code: "",
      matched: false,
    }
  }

  // 3. 无区划树且无既有完整编码时，绝不伪造猜测 0100/0101
  return {
    province_code: existingCodes?.province_code || "",
    city_code: existingCodes?.city_code || "",
    district_code: existingCodes?.district_code || "",
    matched: false,
  }
}

/**
 * 校验当前账号已有地址列表中是否已存在完全一致的地址（防重复插入）
 */
export function findMatchingAddress(
  addresses: AddressResponse[],
  target: {
    name: string
    mobile: string
    province_code: string
    city_code: string
    district_code: string
    address: string
  }
): AddressResponse | undefined {
  if (!Array.isArray(addresses) || addresses.length === 0) return undefined

  const normName = (target.name || "").trim()
  const normMobile = (target.mobile || "").trim()
  const normPCode = (target.province_code || "").trim()
  const normCCode = (target.city_code || "").trim()
  const normDCode = (target.district_code || "").trim()
  const normAddress = (target.address || "").trim()

  return addresses.find((addr) => {
    return (
      (addr.name || "").trim() === normName &&
      (addr.mobile || "").trim() === normMobile &&
      (addr.province_code || "").trim() === normPCode &&
      (addr.city_code || "").trim() === normCCode &&
      (addr.district_code || "").trim() === normDCode &&
      (addr.address || "").trim() === normAddress
    )
  })
}

/**
 * 智能解析整段收货地址文本（例如: "张三，13800138000，北京市海淀区中关村南大街1号"）
 */
export function parseAddressText(rawText: string, tree?: RegionItem[]): ParsedAddress {
  let text = (rawText || "").trim()

  const result: ParsedAddress = {
    name: "",
    mobile: "",
    province: "",
    city: "",
    district: "",
    address: "",
    province_code: "",
    city_code: "",
    district_code: "",
    matched: false,
  }

  if (!text) return result

  // 1. 匹配 11 位手机号码
  const phoneMatch = text.match(/(?:\+?86)?\s*(1[3-9]\d{9})/)
  if (phoneMatch) {
    result.mobile = phoneMatch[1]
    text = text.replace(phoneMatch[0], " ").trim()
  }

  // 2. 若传入了区划树，根据区划树优先提取已知省市区
  const activeTree = tree || getCachedRegionTree()
  if (activeTree && activeTree.length > 0) {
    for (const p of activeTree) {
      const pClean = cleanRegionName(p.name)
      if (text.includes(p.name) || (pClean.length >= 2 && text.includes(pClean))) {
        result.province = p.name
        text = text.replace(p.name, " ").replace(pClean, " ").trim()

        const pChildren = p.children || []
        const isMuni = pChildren.length === 1 && pChildren[0].name === "市辖区"
        if (isMuni) {
          result.city = p.name
          const subDistricts = pChildren[0].children || []
          for (const d of subDistricts) {
            const dClean = cleanRegionName(d.name)
            if (text.includes(d.name) || (dClean.length >= 2 && text.includes(dClean))) {
              result.district = d.name
              text = text.replace(d.name, " ").replace(dClean, " ").trim()
              break
            }
          }
        } else {
          for (const c of pChildren) {
            const cClean = cleanRegionName(c.name)
            if (text.includes(c.name) || (cClean.length >= 2 && text.includes(cClean))) {
              result.city = c.name
              text = text.replace(c.name, " ").replace(cClean, " ").trim()

              for (const d of c.children || []) {
                const dClean = cleanRegionName(d.name)
                if (text.includes(d.name) || (dClean.length >= 2 && text.includes(dClean))) {
                  result.district = d.name
                  text = text.replace(d.name, " ").replace(dClean, " ").trim()
                  break
                }
              }
              break
            }
          }
        }
        break
      }
    }
  }

  // 3. 正则辅助提取未识别的省/市/区
  if (!result.province) {
    const pMatch = text.match(/([^\s,，。]+?(?:省|自治区|特别行政区|市))/)
    if (pMatch) {
      result.province = pMatch[1].trim()
      text = text.replace(pMatch[0], " ").trim()
    }
  }

  if (!result.city) {
    if (result.province && MUNICIPALITIES.some((m) => result.province.includes(m.slice(0, 2)))) {
      result.city = result.province
    } else {
      const cMatch = text.match(/([^\s,，。]+?(?:市|自治州|地区|盟))/)
      if (cMatch) {
        result.city = cMatch[1].trim()
        text = text.replace(cMatch[0], " ").trim()
      }
    }
  }

  if (!result.district) {
    const dMatch = text.match(/([^\s,，。]+?(?:区|县|市|旗|林区))/)
    if (dMatch) {
      result.district = dMatch[1].trim()
      text = text.replace(dMatch[0], " ").trim()
    }
  }

  // 4. 清洗剩余片段，提取姓名与详细地址
  const parts = text
    .split(/[,，\s\n\r]+/)
    .map((s) => s.trim())
    .filter(Boolean)

  if (parts.length > 0) {
    const nameIdx = parts.findIndex((p) => p.length >= 2 && p.length <= 6 && !/\d/.test(p))
    if (nameIdx !== -1) {
      result.name = parts[nameIdx]
      parts.splice(nameIdx, 1)
    }
    result.address = parts.join(" ")
  }

  // 5. 解析并补充编码
  const resolved = resolveRegionCodes(result.province, result.city, result.district, undefined, activeTree || undefined)
  result.province_code = resolved.province_code
  result.city_code = resolved.city_code
  result.district_code = resolved.district_code
  result.matched = resolved.matched

  return result
}
