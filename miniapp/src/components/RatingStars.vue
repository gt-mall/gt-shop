<template>
  <view class="rating-stars" :class="{ interactive }">
    <view class="stars-list">
      <text
        v-for="star in max"
        :key="star"
        class="star-item"
        :class="{ active: star <= currentScore }"
        :style="{ fontSize: size + 'rpx' }"
        @tap="handleSelect(star)"
      >
        {{ star <= currentScore ? '★' : '☆' }}
      </text>
    </view>
    <text v-if="showScore" class="score-text" :style="{ fontSize: (size * 0.8) + 'rpx' }">
      {{ currentScore }}分
    </text>
  </view>
</template>

<script setup lang="ts">
import { computed } from "vue"

interface Props {
  value?: number
  modelValue?: number
  max?: number
  size?: number
  interactive?: boolean
  showScore?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  value: 5,
  modelValue: undefined,
  max: 5,
  size: 28,
  interactive: false,
  showScore: false,
})

const emit = defineEmits<{
  (e: "update:modelValue", val: number): void
  (e: "change", val: number): void
}>()

const currentScore = computed(() => {
  if (props.modelValue !== undefined) {
    return props.modelValue
  }
  return props.value || 0
})

function handleSelect(star: number) {
  if (!props.interactive) return
  emit("update:modelValue", star)
  emit("change", star)
}
</script>

<style lang="scss" scoped>
.rating-stars {
  display: inline-flex;
  align-items: center;
  gap: 8rpx;

  &.interactive {
    cursor: pointer;

    .star-item {
      padding: 0 4rpx;
      transition: transform 0.15s ease;

      &:active {
        transform: scale(1.2);
      }
    }
  }

  .stars-list {
    display: flex;
    align-items: center;
    line-height: 1;
  }

  .star-item {
    color: #e0e0e0;
    transition: color 0.2s ease;
    user-select: none;

    &.active {
      color: #ff9800;
    }
  }

  .score-text {
    color: #ff9800;
    font-weight: 600;
    line-height: 1;
    margin-left: 6rpx;
  }
}
</style>
