import { http } from '@/services/request'

export interface HelloData {
  text: string
}

export interface BootstrapShop {
  name: string
  logo: string
  description: string
  service_name: string
  service_phone: string
  service_wechat: string
  service_wechat_qrcode: string
}

export interface BootstrapSite {
  title: string
  keywords: string
  description: string
  icp: string
  copyright: string
}

export interface BootstrapMp {
  name: string
  original_id: string
  qrcode: string
}

export interface BootstrapCategory {
  id: number
  parent_id: number
  name: string
  image_url: string
  sort: number
}

export interface BootstrapData {
  shop: BootstrapShop
  site: BootstrapSite
  mp: BootstrapMp
  categories: BootstrapCategory[]
}

export interface RegionItem {
  code: string
  name: string
  children?: RegionItem[]
}

export interface RegionsData {
  version: string
  regions: RegionItem[]
}

/**
 * 获取 Hello World 数据
 */
export async function getHello() {
  return await http.get<HelloData>('/api/miniapp/hello')
}

/**
 * 获取小程序/H5 公共启动配置
 */
export async function getBootstrap() {
  return await http.get<BootstrapData>('/api/miniapp/bootstrap')
}

/**
 * 获取全国省市区三级行政区划树
 */
export async function getRegions() {
  return await http.get<RegionsData>('/api/miniapp/regions')
}
