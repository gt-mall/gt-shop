/**
 * 智能收货地址文本解析与区划代码辅助工具
 * 用于拼多多风格的“粘贴文本一键识别”与地址数据规范化
 */

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
}

// 常见直辖市与特区
const MUNICIPALITIES = ["北京市", "上海市", "天津市", "重庆市"]

// 常见省份代码前缀表 (前2位国标码)
const PROVINCE_CODE_MAP: Record<string, string> = {
  北京: "110000",
  天津: "120000",
  河北: "130000",
  山西: "140000",
  内蒙古: "150000",
  辽宁: "210000",
  吉林: "220000",
  黑龙江: "230000",
  上海: "310000",
  江苏: "320000",
  浙江: "330000",
  安徽: "340000",
  福建: "350000",
  江西: "360000",
  山东: "370000",
  河南: "410000",
  湖北: "420000",
  湖南: "430000",
  广东: "440000",
  广西: "450000",
  海南: "460000",
  重庆: "500000",
  四川: "510000",
  贵州: "520000",
  云南: "530000",
  西藏: "540000",
  陕西: "610000",
  甘肃: "620000",
  青海: "630000",
  宁夏: "640000",
  新疆: "650000",
  香港: "810000",
  澳门: "820000",
  台湾: "710000",
}

/**
 * 根据省市区名称兜底生成标准行政区划代码
 */
export function resolveRegionCodes(
  province: string,
  city?: string,
  district?: string,
  existingCodes?: { province_code?: string; city_code?: string; district_code?: string },
): { province_code: string; city_code: string; district_code: string } {
  if (
    existingCodes?.province_code &&
    existingCodes?.city_code &&
    existingCodes?.district_code
  ) {
    return {
      province_code: existingCodes.province_code,
      city_code: existingCodes.city_code,
      district_code: existingCodes.district_code,
    }
  }

  let pCode = "110000"
  for (const [name, code] of Object.entries(PROVINCE_CODE_MAP)) {
    if (province && province.includes(name)) {
      pCode = code
      break
    }
  }

  const prefix = pCode.slice(0, 2)
  const isMunicipality = MUNICIPALITIES.some((m) => province && province.includes(m.slice(0, 2)))

  const cCode = isMunicipality ? `${prefix}0100` : `${prefix}0100`
  const dCode = isMunicipality ? `${prefix}0101` : `${prefix}0101`

  return {
    province_code: existingCodes?.province_code || pCode,
    city_code: existingCodes?.city_code || cCode,
    district_code: existingCodes?.district_code || dCode,
  }
}

/**
 * 智能解析整段收货地址文本
 * 例如: "张三，13800138000，北京市海淀区中关村南大街1号"
 */
export function parseAddressText(rawText: string): ParsedAddress {
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
  }

  if (!text) return result

  // 1. 匹配 11 位手机号码 (以 1 开头，兼容 +86 等前缀)
  const phoneMatch = text.match(/(?:\+?86)?\s*(1[3-9]\d{9})/)
  if (phoneMatch) {
    result.mobile = phoneMatch[1]
    text = text.replace(phoneMatch[0], " ").trim()
  }

  // 2. 匹配常见省份/直辖市
  const provinceList = Object.keys(PROVINCE_CODE_MAP)
  for (const pName of provinceList) {
    const pRegex = new RegExp(`(${pName}(?:省|市|维吾尔自治区|壮族自治区|回族自治区|自治区)?)`, "g")
    const match = pRegex.exec(text)
    if (match) {
      result.province = match[1]
      if (!result.province.endsWith("省") && !result.province.endsWith("市") && !result.province.endsWith("区")) {
        result.province += provinceList.slice(0, 4).includes(pName) ? "市" : "省"
      }
      text = text.replace(match[0], " ")
      break
    }
  }

  // 直辖市特殊处理 (北京市、上海市、天津市、重庆市)
  if (result.province && MUNICIPALITIES.some((m) => result.province.includes(m.slice(0, 2)))) {
    result.city = result.province
  } else {
    // 匹配城市 (以市/地区/州结尾)
    const cityMatch = text.match(/([^\s,，。]+?(?:市|自治州|地区|盟))/)
    if (cityMatch) {
      result.city = cityMatch[1].trim()
      text = text.replace(cityMatch[0], " ")
    }
  }

  // 匹配区县 (以区/县/市/旗结尾)
  const districtMatch = text.match(/([^\s,，。]+?(?:区|县|市|旗))/)
  if (districtMatch) {
    result.district = districtMatch[1].trim()
    text = text.replace(districtMatch[0], " ")
  }

  // 清洗剩余片段
  const parts = text
    .split(/[,，\s\n\r]+/)
    .map((s) => s.trim())
    .filter(Boolean)

  if (parts.length > 0) {
    // 短片段优先作为姓名 (2~6 字符)
    const nameIdx = parts.findIndex((p) => p.length >= 2 && p.length <= 6 && !/\d/.test(p))
    if (nameIdx !== -1) {
      result.name = parts[nameIdx]
      parts.splice(nameIdx, 1)
    }

    // 剩余片段拼接为详细地址
    result.address = parts.join(" ")
  }

  // 补齐行政区划代码
  const codes = resolveRegionCodes(result.province, result.city, result.district)
  result.province_code = codes.province_code
  result.city_code = codes.city_code
  result.district_code = codes.district_code

  return result
}

