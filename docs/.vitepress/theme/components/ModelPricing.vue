<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  DEEPSEEK_PRICE_PERIODS,
  EXCHANGE_RATE,
  IMAGE_GROUP,
  IMAGE_MODELS,
  MODEL_CATEGORIES,
  TEXT_GROUPS,
  calculateImagePriceCny,
  calculateTextPrice,
  formatCny,
  getTextModelsForGroup,
  getEquivalentDiscount,
} from './model-pricing-data.mjs'

const activeCategory = ref(MODEL_CATEGORIES[0].id)
const activeGroupId = ref(MODEL_CATEGORIES[0].defaultGroupId ?? MODEL_CATEGORIES[0].groupIds[0])
const priceMode = ref('group')
const copiedModel = ref('')

onMounted(() => {
  document.documentElement.classList.add('model-pricing-active')
})

onBeforeUnmount(() => {
  document.documentElement.classList.remove('model-pricing-active')
})

const activeCategoryConfig = computed(
  () => MODEL_CATEGORIES.find((category) => category.id === activeCategory.value) ?? MODEL_CATEGORIES[0],
)

const activeGroups = computed(() =>
  activeCategoryConfig.value.groupIds
    .map((groupId) => TEXT_GROUPS.find((group) => group.id === groupId))
    .filter(Boolean),
)

const isImageCategory = computed(() => activeCategoryConfig.value.kind === 'image')

const activeGroup = computed(
  () => activeGroups.value.find((group) => group.id === activeGroupId.value) ?? activeGroups.value[0] ?? TEXT_GROUPS[0],
)

const activeCurrency = computed(() => activeGroup.value.currency ?? 'usd')
const isRmbTextCategory = computed(() => !isImageCategory.value && activeCurrency.value === 'cny')
const isDeepSeekCategory = computed(() => activeCategoryConfig.value.id === 'deepseek')
const isGptCategory = computed(() => activeCategoryConfig.value.id === 'gpt')

const activeTextModels = computed(() => getTextModelsForGroup(activeGroupId.value))

const textRows = computed(() =>
  activeTextModels.value.flatMap((model) => {
    const createRow = (officialPrice, priceLabel = '') => {
      const prices = calculateTextPrice(
        officialPrice,
        activeGroup.value.multiplier,
        activeCurrency.value,
        model.billingOverridesUsd,
      )

      return {
        ...model,
        officialPrice,
        priceLabel,
        priceCurrency: activeCurrency.value,
        prices,
        savingsPercent: Math.round((1 - prices.group.total / prices.official.total) * 100),
      }
    }

    if (isDeepSeekCategory.value && model.officialPeakCny) {
      return [
        createRow(model.officialCny, DEEPSEEK_PRICE_PERIODS.offPeak),
        createRow(model.officialPeakCny, DEEPSEEK_PRICE_PERIODS.peak),
      ]
    }

    if (model.officialLongContextUsd) {
      const priceTiers = [
        createRow(model.officialUsd, '输入 ≤ 10 万 tokens'),
        createRow(model.officialLongContextUsd, '输入 > 10 万 tokens'),
      ]
      return [{ ...priceTiers[0], priceLabel: '', priceTiers }]
    }

    return [createRow(model.officialCny ?? model.officialUsd)]
  }),
)

const pricingRuleExample = computed(() => {
  if (isImageCategory.value) {
    const imageModel = IMAGE_MODELS.find((model) => model.id === 'gpt-image-2.5-flare')
    return `示例：GPT Image 2.5 Flare 当前分组价 ${formatCny(calculateImagePriceCny(imageModel.groupCnyPerImage))} / 张`
  }

  const exampleModel = activeTextModels.value[0]
  const officialPrice = exampleModel.officialCny ?? exampleModel.officialUsd
  const examplePrice = calculateTextPrice(
    officialPrice,
    activeGroup.value.multiplier,
    activeCurrency.value,
    exampleModel.billingOverridesUsd,
  )

  return `示例：${exampleModel.name} 输入官方 ${formatCny(examplePrice.official.input)}，${activeGroup.value.name} 输入价 ${formatCny(examplePrice.group.input)}`
})

const setCategory = (categoryId) => {
  const category = MODEL_CATEGORIES.find((item) => item.id === categoryId)
  if (!category) return

  activeCategory.value = category.id
  activeGroupId.value = category.defaultGroupId ?? category.groupIds[0] ?? ''
  priceMode.value = 'group'
}

const copyModelId = async (modelId) => {
  if (!navigator?.clipboard) return
  await navigator.clipboard.writeText(modelId)
  copiedModel.value = modelId
  window.setTimeout(() => {
    if (copiedModel.value === modelId) copiedModel.value = ''
  }, 1400)
}

const formatUsd = (value) => value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')

const hasBillingOverride = (model, field) => model.billingOverridesUsd?.[field] != null

</script>

<template>
  <main class="model-pricing-page">
    <header class="model-pricing-heading">
      <div>
        <h1>模型价格</h1>
      </div>
    </header>

    <nav class="model-category-tabs" aria-label="模型类型">
      <button
        v-for="category in MODEL_CATEGORIES"
        :key="category.id"
        type="button"
        :class="{ 'is-active': activeCategory === category.id }"
        :aria-pressed="activeCategory === category.id"
        @click="setCategory(category.id)"
      >
        <img
          v-if="category.iconSrc"
          :class="['model-category-icon', `model-category-icon--${category.id}`]"
          :src="category.iconSrc"
          alt=""
          width="25"
          height="25"
        >
        <svg
          v-else-if="category.iconSvg"
          :class="['model-category-icon', `model-category-icon--${category.id}`]"
          viewBox="0 0 24 24"
          aria-hidden="true"
          v-html="category.iconSvg"
        ></svg>
        <span v-else :class="['model-category-mark', `model-category-mark--${category.id}`]">{{ category.mark }}</span>
        {{ category.name }}
      </button>
    </nav>

    <section class="pricing-rule" aria-label="计价规则">
      <div>
        <strong>计价规则</strong>
        <span v-if="isDeepSeekCategory">DeepSeek 采用峰谷计价，高峰时段为北京时间周一至周五 09:00-12:00、14:00-18:00，其余为空闲时段</span>
        <span v-else-if="isRmbTextCategory">官方人民币价格直接显示</span>
        <span v-else-if="!isImageCategory">官方美元价格按 $1 = ¥{{ EXCHANGE_RATE }} 换算</span>
        <span v-if="isGptCategory">一般模型按官方价 × 分组倍率；GPT-6 Astra 缓存读取按实测 $2 计费</span>
        <span v-else-if="isRmbTextCategory">分组价格 = 官方人民币价格 × 分组倍率</span>
        <span v-else-if="!isImageCategory">分组价格 = 官方美元价格 × 分组倍率</span>
        <span v-else>生图分组价格按人民币固定价计费</span>
      </div>
      <span class="pricing-rule-example">
        {{ pricingRuleExample }}
      </span>
    </section>

    <section class="pricing-panel">
      <header class="pricing-panel-header">
        <div>
          <span class="pricing-panel-icon" aria-hidden="true">¥</span>
          <h2>价格列表</h2>
        </div>
        <div class="price-mode-wrap">
          <span v-if="!isImageCategory">{{ isRmbTextCategory
            ? (priceMode === 'group' ? (isDeepSeekCategory ? '分组峰谷价格按人民币计算' : '分组价按人民币计算') : (isDeepSeekCategory ? '官方峰谷价格按人民币显示' : '官方价按人民币显示'))
            : (priceMode === 'group' ? '分组价已按人民币计算' : '官方价按固定汇率换算') }}</span>
          <span v-else>分组价按人民币 / 张显示</span>
          <div v-if="!isImageCategory" class="price-mode-switch" role="group" aria-label="价格类型">
            <button
              type="button"
              :class="{ 'is-active': priceMode === 'group' }"
              :aria-pressed="priceMode === 'group'"
              @click="priceMode = 'group'"
            >分组价格</button>
            <button
              type="button"
              :class="{ 'is-active': priceMode === 'official' }"
              :aria-pressed="priceMode === 'official'"
              @click="priceMode = 'official'"
            >官方价格</button>
          </div>
        </div>
      </header>

      <div v-if="!isImageCategory" class="pricing-content">
        <div
          v-if="priceMode === 'group'"
          class="pricing-groups"
          :class="{
            'pricing-groups--single': activeGroups.length === 1,
            'pricing-groups--gemini': activeCategoryConfig.id === 'gemini',
          }"
          aria-label="选择分组"
        >
          <button
            v-for="group in activeGroups"
            :key="group.id"
            type="button"
            class="pricing-group-card"
            :class="{ 'is-active': activeGroupId === group.id }"
            :aria-pressed="activeGroupId === group.id"
            @click="activeGroupId = group.id"
          >
            <span class="pricing-group-title">
              <strong>{{ group.name }}</strong>
              <em>{{ getEquivalentDiscount(group.multiplier, group.currency) }}</em>
            </span>
            <span>{{ group.multiplier }} 倍率</span>
          </button>
        </div>

        <div
          v-else
          class="pricing-groups pricing-groups--placeholder"
          :class="{
            'pricing-groups--single': activeGroups.length === 1,
            'pricing-groups--gemini': activeCategoryConfig.id === 'gemini',
          }"
          aria-hidden="true"
        >
          <div
            v-for="group in activeGroups"
            :key="`placeholder-${group.id}`"
            class="pricing-group-card pricing-group-placeholder"
          >
            <span class="pricing-group-title">
              <strong>{{ group.name }}</strong>
              <em>官方</em>
            </span>
            <span>官方价格模式</span>
          </div>
        </div>

        <div class="pricing-description">
          <template v-if="priceMode === 'group'">
            <strong>分组介绍：</strong>
            <span>{{ activeGroup.name }}，{{ activeGroup.description }}。</span>
          </template>
          <template v-else>
            <strong>官方价格：</strong>
            <span v-if="isDeepSeekCategory">同时显示官方峰谷价格；高峰时段为北京时间周一至周五 09:00-12:00、14:00-18:00，其余为空闲时段。</span>
            <span v-else-if="isRmbTextCategory">按人民币官方基准显示，仅用于和分组价格对比。</span>
            <span v-else>按 $1 = ¥{{ EXCHANGE_RATE }} 折算成人民币，仅用于和分组价格对比。</span>
          </template>
        </div>

        <div class="pricing-table-scroll">
          <table
            class="pricing-table"
            :class="{ 'pricing-table--deepseek': isDeepSeekCategory }"
          >
            <thead>
              <tr>
                <th scope="col">模型 ID</th>
                <th v-if="isDeepSeekCategory" scope="col">计费时段</th>
                <th scope="col">输入价格</th>
                <th scope="col">输出价格</th>
                <th scope="col">缓存读取</th>
                <th scope="col">输入 + 输出合计</th>
                <th scope="col">节省幅度</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="model in textRows"
                :key="`${model.id}-${model.priceLabel || 'default'}`"
                :class="{ 'is-featured-model': model.featured }"
              >
                <td>
                  <div class="model-id-cell">
                    <div>
                      <div class="model-id-title">
                        <strong>{{ model.id }}</strong>
                        <span v-if="model.featured" class="featured-model-badge">
                          {{ model.featuredLabel ?? '主推' }}
                        </span>
                        <span v-if="model.billingOverridesUsd" class="billing-model-badge">缓存读取按实测</span>
                      </div>
                      <span>{{ model.description }}</span>
                      <span v-if="model.priceTiers" class="model-context-note">
                        按单次请求的输入 tokens 数分档；输入、输出和缓存读取均按对应档计费。
                      </span>
                    </div>
                    <button
                      type="button"
                      class="copy-model-button"
                      :aria-label="`复制 ${model.id}`"
                      :title="copiedModel === model.id ? '已复制' : '复制模型 ID'"
                      @click="copyModelId(model.id)"
                    >
                      <span aria-hidden="true"></span>
                    </button>
                  </div>
                </td>
                <template v-if="priceMode === 'group' && model.unavailableMessage">
                  <td :colspan="isDeepSeekCategory ? 6 : 5" class="model-unavailable-cell">
                    {{ model.unavailableMessage }}
                  </td>
                </template>
                <template v-else>
                  <td v-if="isDeepSeekCategory" class="deepseek-price-period">
                    <strong>{{ model.priceLabel }}</strong>
                  </td>
                  <td v-for="field in ['input', 'output', 'cachedInput', 'total']" :key="field">
                    <div
                      v-for="tier in model.priceTiers ?? [model]"
                      :key="tier.priceLabel || 'default'"
                      :class="{ 'context-price-tier': model.priceTiers }"
                    >
                      <span v-if="model.priceTiers" class="context-tier-label">{{ tier.priceLabel }}</span>
                      <template v-if="priceMode === 'group'">
                        <strong class="group-price">{{ formatCny(tier.prices.group[field]) }}</strong>
                        <span class="price-unit">/ 1M tokens</span>
                        <span
                          v-if="hasBillingOverride(tier, field) && tier.priceCurrency === 'usd'"
                          class="astra-price-basis"
                        >实测基准 ${{ formatUsd(tier.billingOverridesUsd[field]) }} × {{ activeGroup.multiplier }}</span>
                        <del v-else>{{ tier.priceBasisLabel ?? '官方' }} {{ formatCny(tier.prices.official[field]) }}</del>
                      </template>
                      <template v-else>
                        <strong class="official-price">{{ formatCny(tier.prices.official[field]) }}</strong>
                        <span class="price-unit">/ 1M tokens</span>
                        <span v-if="tier.priceCurrency === 'usd'" class="official-usd">
                          ${{ field === 'total'
                            ? (tier.officialPrice.input + tier.officialPrice.output).toFixed(2)
                            : tier.officialPrice[field].toFixed(3).replace(/0+$/, '').replace(/\.$/, '') }}
                        </span>
                        <span v-else class="official-cny">人民币基准</span>
                      </template>
                    </div>
                  </td>
                  <td>
                    <span v-if="priceMode === 'group'" class="saving-badge">
                      省 {{ model.savingsPercent }}%
                    </span>
                    <span v-else class="official-label">
                      {{ model.priceBasisLabel ?? (model.billingOverridesUsd ? '官方公开价' : '官方基准') }}
                    </span>
                  </td>
                </template>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="isGptCategory" class="astra-pricing-note">
          <strong>GPT-6 Astra 计费说明：</strong>
          输入和输出按官方公开价 $10、$50 计算；缓存读取的官方公开价为 $1，但社区长期观察并经大量用户实际调用验证，实际按 $2 计费。本站三个 GPT 分组的 Astra 缓存读取因此按实测基准 $2 乘对应分组倍率计算。官方价格或计费规则发生变化后，本站会尽快同步调整。
        </p>
      </div>

      <div v-else class="pricing-content">
        <div v-if="priceMode === 'group'" class="pricing-groups pricing-groups--image">
          <div class="pricing-group-card pricing-group-static is-active">
            <span class="pricing-group-title">
              <strong>{{ IMAGE_GROUP.name }}</strong>
              <em>按模型计价</em>
            </span>
          </div>
        </div>

        <div class="pricing-description">
          <strong>绘图价格：</strong>
          <span>当前分组按模型设置默认人民币价格，图片请求按张计费。</span>
        </div>

        <div class="pricing-table-scroll">
          <table class="pricing-table pricing-table--image">
            <thead>
              <tr>
                <th scope="col">模型 ID</th>
                <th scope="col">模型介绍</th>
                <th scope="col">接口 / 规格</th>
                <th scope="col">当前分组价（人民币）</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="model in IMAGE_MODELS" :key="model.id">
                <td>
                  <div class="model-id-cell">
                    <div>
                      <strong
                        v-if="model.recommended"
                        class="image-model-id--recommended"
                      >{{ model.id }}</strong>
                      <span v-else class="image-model-id">{{ model.id }}</span>
                      <span v-if="model.recommended">最新推荐模型</span>
                    </div>
                    <button
                      type="button"
                      class="copy-model-button"
                      :aria-label="`复制 ${model.id}`"
                      :title="copiedModel === model.id ? '已复制' : '复制模型 ID'"
                      @click="copyModelId(model.id)"
                    ><span aria-hidden="true"></span></button>
                  </div>
                </td>
                <td><span class="model-description">{{ model.description }}</span></td>
                <td>
                  <strong class="table-plain-value">{{ model.route }}</strong>
                  <span class="table-subvalue">{{ model.spec }}</span>
                </td>
                <td>
                  <template v-if="model.sizePricesCny">
                    <div class="image-size-prices">
                      <span
                        v-for="sizePrice in model.sizePricesCny"
                        :key="sizePrice.size"
                        class="image-size-price"
                      >
                        <span class="image-size-label">{{ sizePrice.size }}</span>
                        <strong class="group-price">{{ formatCny(calculateImagePriceCny(sizePrice.cnyPerImage)) }}</strong>
                        <span class="price-unit">/ 张</span>
                      </span>
                    </div>
                    <span class="table-subvalue">按分辨率计费</span>
                  </template>
                  <template v-else>
                    <strong class="group-price">{{ formatCny(calculateImagePriceCny(model.groupCnyPerImage)) }}</strong>
                    <span class="price-unit">/ 张</span>
                    <span class="table-subvalue">{{ model.priceNote ?? '当前分组默认价' }}</span>
                  </template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

  </main>
</template>

<style>
:root.model-pricing-active,
:root.model-pricing-active body,
:root.model-pricing-active .Layout {
  max-width: 100vw;
  overflow-x: hidden;
}

:root.model-pricing-active .VPNav,
:root.model-pricing-active .VPNavBar,
:root.model-pricing-active .VPNavScreen,
:root.model-pricing-active .VPLocalNav,
:root.model-pricing-active .VPSidebar {
  display: none !important;
}

:root.model-pricing-active .VPContent,
:root.model-pricing-active .VPContent.has-sidebar {
  padding-left: 0 !important;
}

:root.model-pricing-active .VPDoc,
.VPDoc:has(.model-pricing-page) {
  box-sizing: border-box;
  max-width: 100vw;
  overflow-x: hidden;
  padding: 32px var(--site-shell-inset, 32px) 56px !important;
}

:root.model-pricing-active .VPDoc .container,
:root.model-pricing-active .VPDoc .content-container,
.VPDoc:has(.model-pricing-page) .container,
.VPDoc:has(.model-pricing-page) .content-container {
  width: 100% !important;
  max-width: none !important;
}

:root.model-pricing-active .VPDoc .content,
.VPDoc:has(.model-pricing-page) .content {
  width: 100% !important;
  max-width: none !important;
  margin: 0 !important;
}

:root.model-pricing-active .VPDoc .container,
.VPDoc:has(.model-pricing-page) .container {
  padding-top: 0 !important;
}

:root.model-pricing-active .VPDoc .vp-doc,
.VPDoc:has(.model-pricing-page) .vp-doc {
  font-size: 17px;
  line-height: 1.65;
}

:root.model-pricing-active .VPDoc .prev-next,
.VPDoc:has(.model-pricing-page) .prev-next {
  display: none;
}

@media (min-width: 960px) {
  :root.model-pricing-active body {
    --site-shell-inset: clamp(24px, 2vw, 64px);
    --site-sidebar-left: max(0px, calc(var(--site-shell-inset) - 72px));
  }
}

.model-pricing-page {
  --pricing-active-bg: #f3ad72;
  --pricing-active-border: #e78e45;
  --pricing-active-text: #3f2818;
  --pricing-active-chip-bg: rgba(255, 250, 245, 0.94);
  --pricing-active-chip-text: #9f4a17;
  --pricing-active-shadow: 0 4px 10px rgba(170, 84, 22, 0.1), inset 0 0 0 1px rgba(255, 250, 245, 0.28);
  min-width: 0;
  max-width: 100%;
  width: 100%;
  color: var(--vp-c-text-1);
  font-family: var(--vp-font-family-base);
  letter-spacing: 0;
}

.model-pricing-page button {
  font-family: inherit;
  letter-spacing: 0;
}

.model-pricing-heading {
  display: flex;
  min-height: 0;
  align-items: flex-start;
  justify-content: space-between;
  padding: 0 0 24px;
}

.model-pricing-heading h1 {
  margin: 0 0 16px;
  border: 0;
  color: var(--vp-c-text-1);
  font-family: var(--vp-font-family-base);
  font-size: 48px;
  font-weight: 750;
  line-height: 1.28;
}

.model-category-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  min-height: 68px;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-surface);
  box-shadow: 0 4px 14px rgba(108, 63, 31, 0.035);
}

.model-category-tabs button {
  display: flex;
  min-width: 0;
  height: 52px;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 9px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--vp-c-text-2);
  font-size: 22px;
  font-weight: 560;
  white-space: nowrap;
  cursor: pointer;
}

.model-category-tabs button.is-active {
  border-color: var(--pricing-active-border);
  background: var(--pricing-active-bg);
  box-shadow: var(--pricing-active-shadow);
  color: var(--pricing-active-text);
}

.model-category-tabs .model-category-icon {
  display: block;
  width: 25px;
  height: 25px;
  flex: 0 0 25px;
  margin: 0;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  color: currentColor;
  object-fit: contain;
  cursor: inherit;
  transition: none;
}

.model-category-mark {
  display: inline-flex;
  width: 25px;
  height: 25px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #fff;
  font-size: 14px;
  font-weight: 800;
}

.model-category-mark--image { background: #da6b35; }
.model-category-mark--domestic { background: #56758d; }

.model-category-tabs button.is-active .model-category-mark {
  background: var(--pricing-active-chip-bg);
  color: var(--pricing-active-chip-text);
}

.pricing-rule {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-top: 16px;
  padding: 10px 20px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-surface);
  color: var(--vp-c-text-2);
  font-size: 18px;
}

.pricing-rule > div {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
}

.pricing-rule strong {
  color: var(--vp-c-text-1);
  font-size: 18px;
}

.pricing-rule-example {
  flex: 0 1 auto;
  color: var(--vp-c-text-3);
  text-align: right;
}

.pricing-panel {
  min-width: 0;
  max-width: 100%;
  margin-top: 16px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: var(--site-surface);
  box-shadow: 0 5px 18px rgba(108, 63, 31, 0.045);
  overflow: hidden;
}

.pricing-panel-header {
  display: flex;
  min-height: 66px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--site-line);
}

.pricing-panel-header > div:first-child {
  display: flex;
  align-items: center;
  gap: 10px;
}

.model-pricing-page .pricing-panel-header h2 {
  margin: 0 !important;
  padding: 0 !important;
  border: 0 !important;
  color: var(--vp-c-text-1);
  font-size: 26px;
  font-weight: 750;
  line-height: 1.3;
}

.pricing-panel-icon {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border: 1.5px solid #9a765c;
  border-radius: 50%;
  color: #765941;
  font-size: 13px;
  font-weight: 800;
}

.price-mode-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--vp-c-text-3);
  font-size: 16px;
}

.price-mode-switch {
  display: inline-flex;
  padding: 3px;
  border: 1px solid var(--site-line);
  border-radius: 18px;
  background: #f8eee2;
}

.price-mode-switch button {
  min-width: 76px;
  height: 28px;
  padding: 0 12px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--vp-c-text-2);
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
}

.price-mode-switch button.is-active {
  background: #a64b0d;
  box-shadow: 0 1px 3px rgba(75, 37, 12, 0.18);
  color: #fff;
}

.pricing-content {
  min-width: 0;
  max-width: 100%;
  padding: 20px;
}

.pricing-groups {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(260px, 100%), 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.pricing-groups--single {
  display: grid;
  width: 100%;
  max-width: 100%;
  grid-template-columns: minmax(0, 1fr);
}

.pricing-groups--single.pricing-groups--gemini {
  width: min(680px, 100%);
}

.pricing-group-card {
  display: flex;
  min-height: 82px;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  padding: 14px 16px;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  background: rgba(255, 252, 247, 0.94);
  color: var(--vp-c-text-2);
  font-size: 18px;
  text-align: left;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, color 0.16s ease;
}

.pricing-groups > button { cursor: pointer; }

.pricing-groups--single > button {
  min-height: 56px;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.pricing-groups--single > button > span:last-child {
  flex: 1 1 auto;
  text-align: right;
}

.pricing-group-card.is-active {
  border-color: var(--pricing-active-border);
  background: var(--pricing-active-bg);
  box-shadow: var(--pricing-active-shadow);
  color: var(--pricing-active-text);
}

.pricing-group-placeholder {
  border-style: dashed;
  background: var(--site-surface-muted);
  color: var(--vp-c-text-3);
  box-shadow: none;
}

.pricing-group-placeholder .pricing-group-title strong {
  color: var(--vp-c-text-2);
}

.pricing-group-placeholder .pricing-group-title em {
  background: #f1e8dd;
  color: var(--vp-c-text-3);
}

.pricing-groups--single > .pricing-group-placeholder {
  min-height: 56px;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.pricing-groups--single > .pricing-group-placeholder > span:last-child {
  flex: 1 1 auto;
  text-align: right;
}

.pricing-group-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.pricing-group-title strong {
  color: var(--vp-c-text-1);
  font-size: 21px;
  font-weight: 750;
}

.pricing-group-card.is-active .pricing-group-title strong {
  color: #fff;
}

.pricing-group-title em {
  padding: 3px 9px;
  border-radius: 12px;
  background: #ffe2d3;
  color: #a84c14;
  font-size: 16px;
  font-style: normal;
  font-weight: 800;
  line-height: 1.4;
  white-space: nowrap;
}

.pricing-group-card.is-active .pricing-group-title em {
  background: var(--pricing-active-chip-bg);
  color: var(--pricing-active-chip-text);
}

.pricing-groups--image {
  display: grid;
  width: min(420px, 100%);
  max-width: 100%;
  grid-template-columns: minmax(0, 1fr);
}

.pricing-groups--image .pricing-group-card {
  min-height: 56px;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.pricing-description {
  display: flex;
  min-height: 48px;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 10px 16px;
  border: 1px solid #f2d8ca;
  border-radius: 8px;
  background: #fffaf7;
  color: var(--vp-c-text-2);
  font-size: 18px;
}

.pricing-description strong {
  flex: 0 0 auto;
  color: #9f4a17;
  font-weight: 750;
}

.pricing-table-scroll {
  width: 100%;
  max-width: 100%;
  border: 1px solid var(--site-line);
  border-radius: 8px;
  overflow-x: auto;
}

.model-pricing-page .pricing-table {
  width: 100%;
  min-width: 1040px;
  margin: 0;
  border: 0;
  border-collapse: collapse;
  border-radius: 0;
  font-size: 17px;
  table-layout: fixed;
}

.model-pricing-page .pricing-table th,
.model-pricing-page .pricing-table td {
  border: 0;
  border-bottom: 1px solid var(--site-line);
  text-align: left;
  vertical-align: middle;
}

.model-pricing-page .pricing-table th {
  height: 48px;
  padding: 0 16px;
  background: #f8eee2;
  color: var(--vp-c-text-2);
  font-size: 17px;
  font-weight: 650;
}

.model-pricing-page .pricing-table td {
  height: 86px;
  padding: 12px 16px;
  background: rgba(255, 252, 247, 0.94);
  color: var(--vp-c-text-1);
}

.model-pricing-page .pricing-table td.model-unavailable-cell {
  color: #9f4a17;
  font-size: 18px;
  font-weight: 750;
  text-align: center;
}

.model-pricing-page .pricing-table tbody tr.is-featured-model td {
  border-bottom-color: #d9cff4;
  background: #f5f1ff;
}

.model-pricing-page .pricing-table tbody tr.is-featured-model td:first-child {
  box-shadow: inset 4px 0 0 #8069b3;
}

.model-pricing-page .pricing-table tbody tr:last-child td { border-bottom: 0; }
.model-pricing-page .pricing-table th:first-child { width: 23%; }
.model-pricing-page .pricing-table th:nth-child(2),
.model-pricing-page .pricing-table th:nth-child(3),
.model-pricing-page .pricing-table th:nth-child(4) { width: 15%; }
.model-pricing-page .pricing-table th:nth-child(5) { width: 18%; }
.model-pricing-page .pricing-table th:nth-child(6) { width: 14%; }

.model-pricing-page .pricing-table--deepseek th:first-child { width: 21%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(2) { width: 20%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(3) { width: 11%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(4) { width: 12%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(5) { width: 12%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(6) { width: 14%; }
.model-pricing-page .pricing-table--deepseek th:nth-child(7) { width: 10%; }

.model-pricing-page .pricing-table--image {
  min-width: 980px;
}

.model-pricing-page .pricing-table--image th:first-child { width: 24%; }
.model-pricing-page .pricing-table--image th:nth-child(2) { width: 34%; }
.model-pricing-page .pricing-table--image th:nth-child(3) { width: 20%; }
.model-pricing-page .pricing-table--image th:nth-child(4) { width: 22%; }

.model-id-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.deepseek-price-period {
  min-width: 210px;
  color: #7a3f20;
  font-size: 15px;
  line-height: 1.55;
}

.deepseek-price-period strong { font-weight: 750; }

.model-id-cell > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.model-id-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.featured-model-badge {
  display: inline-flex;
  min-width: 0;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  padding: 2px 7px;
  border-radius: 10px;
  background: #8069b3;
  color: #fff !important;
  font-size: 13px;
  font-weight: 750;
  line-height: 1.35;
}

.model-id-cell strong {
  color: var(--vp-c-text-1);
  font-size: 18px;
  font-weight: 750;
}

.model-id-cell > div > span {
  color: var(--vp-c-text-3);
  font-size: 16px;
  line-height: 1.35;
}

.model-id-cell > div > .model-context-note {
  margin-top: 6px;
  color: var(--vp-c-text-2);
  font-size: 14px;
  line-height: 1.5;
}

.context-price-tier + .context-price-tier {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--vp-c-divider);
}

.context-tier-label {
  display: block;
  margin-bottom: 4px;
  color: var(--vp-c-text-2);
  font-size: 14px;
  font-weight: 700;
  white-space: nowrap;
}

.model-id-cell .image-model-id {
  color: var(--vp-c-text-1);
  font-size: 18px;
  font-weight: 600;
}

.model-id-cell .image-model-id--recommended {
  color: #2f6fca;
}

.copy-model-button {
  position: relative;
  width: 24px;
  height: 24px;
  flex: 0 0 24px;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.copy-model-button::before,
.copy-model-button span {
  content: '';
  position: absolute;
  width: 8px;
  height: 8px;
  border: 1.5px solid #ad9a89;
  border-radius: 2px;
}

.copy-model-button::before { top: 6px; left: 7px; }
.copy-model-button span { top: 9px; left: 10px; background: #fffaf4; }
.copy-model-button:hover::before,
.copy-model-button:hover span { border-color: #a64b0d; }

.group-price,
.official-price {
  color: #ad500f;
  font-size: 21px;
  font-weight: 800;
  white-space: nowrap;
}

.official-price { color: var(--vp-c-text-1); }

.price-unit {
  margin-left: 3px;
  color: var(--vp-c-text-3);
  font-size: 14px;
  white-space: nowrap;
}

.image-size-prices {
  display: grid;
  gap: 4px;
}

.image-size-price {
  display: grid;
  grid-template-columns: 28px 64px auto;
  align-items: baseline;
  justify-content: start;
}

.image-size-label {
  color: var(--vp-c-text-2);
  font-size: 14px;
  font-weight: 700;
}

.image-size-price .group-price {
  font-size: 17px;
}

.image-size-price .price-unit {
  margin-left: 0;
}

.model-pricing-page .pricing-table del,
.official-usd {
  display: block;
  margin-top: 3px;
  color: #aa9b8f;
  font-size: 14px;
  line-height: 1.35;
  text-decoration-thickness: 1px;
}

.official-usd { text-decoration: none; }

.official-cny {
  display: block;
  margin-top: 3px;
  color: #aa9b8f;
  font-size: 14px;
  line-height: 1.35;
}

.saving-badge,
.official-label {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  justify-content: center;
  padding: 4px 11px;
  border-radius: 15px;
  background: #eaf9ef;
  color: #239653;
  font-size: 16px;
  font-weight: 800;
  white-space: nowrap;
}

.official-label {
  background: #f3e9de;
  color: var(--vp-c-text-2);
}

.table-plain-value {
  color: var(--vp-c-text-1);
  font-size: 18px;
  font-weight: 650;
}

.table-subvalue,
.model-description {
  display: block;
  color: var(--vp-c-text-3);
  font-size: 15px;
  line-height: 1.5;
}

.model-description {
  color: var(--vp-c-text-2);
}

.astra-pricing-note {
  margin: 12px 0 0 !important;
  padding: 10px 14px;
  border-left: 3px solid #d96f35;
  border-radius: 5px;
  background: #fff8f3;
  color: var(--vp-c-text-2) !important;
  font-size: 15px !important;
  line-height: 1.6 !important;
}

.astra-pricing-note strong {
  color: #9f4a17;
}

.astra-price-basis {
  display: block;
  margin-top: 4px;
  color: #a65a31;
  font-size: 13px;
  line-height: 1.35;
  white-space: nowrap;
}

.billing-model-badge {
  display: inline-flex;
  padding: 2px 7px;
  border-radius: 10px;
  background: #fff0e5;
  color: #a84c14;
  font-size: 12px;
  font-weight: 750;
  white-space: nowrap;
}

.dark .model-pricing-page { color: #f4eee5; }
.dark .model-pricing-heading h1,
.dark .pricing-panel-header h2,
.dark .pricing-group-title strong,
.dark .model-id-cell strong,
.dark .table-plain-value { color: #f4eee5; }
.dark .model-category-tabs,
.dark .pricing-rule,
.dark .pricing-panel,
.dark .pricing-groups > button,
.dark .pricing-group-placeholder,
.dark .pricing-group-static,
.dark .model-pricing-page .pricing-table td { background: #211f1d; border-color: #3a3530; }
.dark .model-pricing-page .pricing-table tbody tr.is-featured-model td {
  border-bottom-color: #55486f;
  background: #292335;
}
.dark .model-pricing-page .pricing-table tbody tr.is-featured-model td:first-child {
  box-shadow: inset 4px 0 0 #a18bd0;
}
.dark .featured-model-badge { background: #a18bd0; color: #21192e !important; }
.dark .deepseek-price-period { color: #e7b58a; }
.dark .model-category-tabs,
.dark .pricing-rule,
.dark .pricing-panel,
.dark .pricing-table-scroll { border-color: #3a3530; }
.dark .model-pricing-page .pricing-table th { background: #2a2724; color: #c2b9b0; }
.dark .pricing-description { border-color: #5a3b29; background: #2b211b; }
.dark .copy-model-button span { background: #211f1d; }
.dark .model-category-icon--hunyuan { filter: invert(1); }
.dark .model-category-tabs button.is-active .model-category-icon--hunyuan { filter: none; }
.dark .model-category-tabs button.is-active,
.dark .pricing-group-card.is-active {
  border-color: #e78e45;
  background: #de8442;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.16), inset 0 0 0 1px rgba(255, 250, 245, 0.2);
  color: #25160d;
}
.dark .pricing-group-card.is-active .pricing-group-title strong { color: #25160d; }
.dark .pricing-group-card.is-active .pricing-group-title em,
.dark .model-category-tabs button.is-active .model-category-mark {
  background: rgba(255, 242, 230, 0.92);
  color: #8c3b0d;
}
.dark .pricing-group-placeholder .pricing-group-title strong { color: #d0c2b5; }
.dark .pricing-group-placeholder .pricing-group-title em { background: #3a3028; color: #9f9083; }
.dark .astra-pricing-note { border-color: #d97745; background: #2b211b; }
.dark .astra-pricing-note strong { color: #f0a06d; }
.dark .astra-price-basis { color: #e6a178; }
.dark .billing-model-badge { background: #4a2d1d; color: #f0a06d; }

@media (max-width: 1100px) {
  .price-mode-wrap > span,
  .pricing-rule-example { display: none; }
}

@media (max-width: 768px) {
  :root.model-pricing-active .VPDoc,
  .VPDoc:has(.model-pricing-page) { padding: 32px 16px 40px !important; }
  :root.model-pricing-active .VPDoc .container,
  .VPDoc:has(.model-pricing-page) .container { padding-top: 0 !important; }
  .model-pricing-heading { min-height: 0; padding-bottom: 20px; }
  .model-pricing-heading h1 { margin-bottom: 16px; font-size: 36px; }
  .model-category-tabs {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    min-height: 0;
  }
  .model-category-tabs button { height: 50px; gap: 8px; padding: 0 10px; font-size: 19px; }
  .model-category-tabs .model-category-icon { width: 22px; height: 22px; flex-basis: 22px; }
  .model-category-mark { width: 22px; height: 22px; font-size: 11px; }
  .pricing-rule { align-items: flex-start; }
  .pricing-rule > div { gap: 4px 14px; }
  .pricing-rule > div span { width: 100%; }
  .pricing-panel-header { align-items: flex-start; flex-direction: column; gap: 10px; }
  .price-mode-wrap { width: 100%; }
  .price-mode-switch { width: 100%; }
  .price-mode-switch button { flex: 1; }
  .pricing-content { padding: 14px; }
  .pricing-groups,
  .pricing-groups--image {
    width: 100%;
    grid-template-columns: minmax(0, 1fr);
  }
  .pricing-groups--single {
    width: 100%;
    grid-template-columns: minmax(0, 1fr);
  }
  .pricing-groups--single > button {
    min-height: 82px;
    flex-direction: column;
    align-items: stretch;
    justify-content: center;
    gap: 6px;
  }
  .pricing-groups--single > .pricing-group-placeholder {
    min-height: 82px;
    flex-direction: column;
    align-items: stretch;
    justify-content: center;
    gap: 6px;
  }
  .pricing-groups--single > button > span:last-child { text-align: left; }
  .pricing-groups--single > .pricing-group-placeholder > span:last-child { text-align: left; }
  .pricing-description { align-items: flex-start; flex-direction: column; gap: 2px; }
  .model-pricing-page .pricing-table { min-width: 980px; }
  .model-pricing-page .pricing-table--image { min-width: 980px; }
}
</style>
