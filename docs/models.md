---
title: 模型价格
navbar: false
sidebar: false
aside: false
outline: false
---

<script setup>
import ModelPricing from './.vitepress/theme/components/ModelPricing.vue'
</script>

<ModelPricing />

GPT-6.2 Astra 和 Sol 的输入、缓存读取与输出参考价来自 [OpenAI 官方公告](https://openai.com/index/introducing-gpt-6-2-astra-sol/)。Gemini 4 Neon 的官方参考价来自 [Google 官方公告](https://blog.google/innovation-and-ai/models-and-research/gemini-models/introducing-gemini-4-neon/)，本站 Neon 分组的缓存读取价待确认。

## 如何切换模型

价格表第一列是模型 ID，点击旁边按钮复制。切换模型时，只修改客户端里的 `Model`；Base URL 和 API Key 保持不变。

完整填写方式看 [客户端接入](/clients/#模型怎么填)。
