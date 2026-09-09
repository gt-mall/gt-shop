<script setup lang="ts">
import { onLaunch, onShow, onHide } from '@dcloudio/uni-app'
import { useBootstrap } from '@/services/bootstrap'

onLaunch(async () => {
  console.log('App Launch')
  const { fetchBootstrap } = useBootstrap()
  await fetchBootstrap()
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
