<script setup lang="ts">
import { computed, ref } from 'vue'

const copied = ref(false)
const failed = ref(false)
const siteUrl = 'https://www.myautoapi.com/'

const shortcut = computed(() => {
  if (typeof navigator === 'undefined') {
    return 'Ctrl+D'
  }

  return /Mac|iPhone|iPad|iPod/i.test(navigator.platform) ? '⌘D' : 'Ctrl+D'
})

const message = computed(() => {
  if (copied.value) {
    return `网址已复制。按 ${shortcut.value} 收藏本站。`
  }

  if (failed.value) {
    return `按 ${shortcut.value} 收藏本站。`
  }

  return '点击复制本站地址，再用浏览器快捷键收藏。'
})

/**
 * 现代浏览器不允许网页直接写入用户收藏夹，所以这里仅复制站点地址并提示快捷键。
 * 这样不会触发被拦截的旧式收藏 API，也能在 Chrome、Safari、Firefox 等浏览器稳定工作。
 */
async function copySiteUrl() {
  copied.value = false
  failed.value = false

  try {
    await navigator.clipboard.writeText(siteUrl)
    copied.value = true
  } catch {
    failed.value = true
  }
}
</script>

<template>
  <section class="contact-bookmark">
    <div>
      <h2>收藏本站</h2>
      <p>{{ message }}</p>
    </div>
    <button type="button" class="contact-bookmark__button" @click="copySiteUrl">
      复制本站地址
    </button>
  </section>
</template>
