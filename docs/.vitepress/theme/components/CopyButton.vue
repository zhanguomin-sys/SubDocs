<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  text: string
  label?: string
}>()

const copied = ref(false)
const failed = ref(false)

const buttonText = computed(() => {
  if (copied.value) {
    return '已复制'
  }

  if (failed.value) {
    return '复制失败'
  }

  return props.label || '复制'
})

/**
 * 只复制文本，不跳转外部应用，避免 Telegram 链接触发浏览器或系统的敏感跳转提示。
 */
async function copyText() {
  copied.value = false
  failed.value = false

  try {
    await navigator.clipboard.writeText(props.text)
    copied.value = true
  } catch {
    failed.value = true
  }
}
</script>

<template>
  <button type="button" class="contact-copy-button" @click="copyText">
    {{ buttonText }}
  </button>
</template>
