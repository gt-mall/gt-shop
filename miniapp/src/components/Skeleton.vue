<template>
  <view
    class="skeleton-block"
    :class="[
      'skeleton-' + type,
      { 'skeleton-animated': animated }
    ]"
    :style="computedStyle"
  />
</template>

<script setup lang="ts">
import { computed } from "vue"

interface Props {
  type?: "rect" | "circle" | "text"
  width?: string | number
  height?: string | number
  radius?: string | number
  animated?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: "rect",
  width: "100%",
  height: "",
  radius: "",
  animated: true,
})

function formatDimension(val?: string | number, fallback = ""): string {
  if (val === undefined || val === null || val === "") return fallback
  if (typeof val === "number") return `${val}rpx`
  return val
}

const computedStyle = computed(() => {
  let defaultHeight = "100rpx"
  let defaultRadius = "12rpx"

  if (props.type === "circle") {
    defaultHeight = "80rpx"
    defaultRadius = "50%"
  } else if (props.type === "text") {
    defaultHeight = "28rpx"
    defaultRadius = "6rpx"
  }

  const w = formatDimension(props.width, props.type === "circle" ? defaultHeight : "100%")
  const h = formatDimension(props.height, defaultHeight)
  const r = formatDimension(props.radius, defaultRadius)

  return {
    width: w,
    height: h,
    borderRadius: r,
  }
})
</script>

<style lang="scss" scoped>
.skeleton-block {
  display: block;
  background-color: #f0f2f5;
  box-sizing: border-box;
  overflow: hidden;
  position: relative;

  &.skeleton-animated {
    &::after {
      content: "";
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: linear-gradient(
        90deg,
        rgba(255, 255, 255, 0) 0%,
        rgba(255, 255, 255, 0.45) 50%,
        rgba(255, 255, 255, 0) 100%
      );
      animation: skeleton-shimmer 1.5s infinite;
    }
  }
}

@keyframes skeleton-shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-block.skeleton-animated::after {
    animation: none;
  }
}
</style>
