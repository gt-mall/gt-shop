import { http } from '@/services/request'

export interface HelloData {
  text: string
}

/**
 * 获取 Hello World 数据
 */
export async function getHello() {
  return await http.get<HelloData>('/api/miniapp/hello')
}
