<script setup lang="ts">
import { onLaunch, onShow, onHide } from '@dcloudio/uni-app'
import { useAppStore } from '@/stores/app'
import { useMemberStore } from '@/stores/member'

onLaunch(async () => {
  console.log('App Launch')
  const appStore = useAppStore()
  const memberStore = useMemberStore()

  // 初始化商城全局公共配置（静默失败，不阻塞启动）
  appStore.init(true).catch((err) => {
    console.warn('[App Launch] appStore.init failed:', err)
  })

  // 监听全局认证失效事件：统一执行 Store 登出与状态清理
  uni.$on('auth-expired', () => {
    console.warn('[App] 收到认证失效通知，执行全局登出')
    memberStore.logout()
  })

  // 若已有有效 token，静默刷新会员资料
  if (memberStore.token) {
    memberStore.refreshMember().catch((err) => {
      console.warn('[App Launch] memberStore.refreshMember failed:', err)
    })
  }
})
onShow(() => {
  console.log('App Show')
})
onHide(() => {
  console.log('App Hide')
})
</script>

<style>
/* 每个页面公共 css */
page {
  background-color: #f8f8f8;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
}
</style>
