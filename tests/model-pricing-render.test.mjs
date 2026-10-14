import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'

test('renders Gemini 4 models first with 0.35 pricing and the Barium channel baseline label', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')

    for (const priceMode of ['group', 'official']) {
      const component = {
        ...pricing,
        setup(props, context) {
          const state = pricing.setup(props, context)
          state.activeCategory.value = 'gemini'
          state.activeGroupId.value = 'gemini-antigravity'
          state.priceMode.value = priceMode
          return state
        },
      }
      const html = await renderToString(createSSRApp(component))
      const rows = [...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
        .map(([row]) => row)
        .filter((row) => row.includes('class="model-id-cell"'))
      const ids = rows.map((row) => row.match(/<strong>(gemini-[^<]+)<\/strong>/)[1])

      assert.equal(ids.length, 16)
      assert.deepEqual(ids.slice(0, 3), ['gemini-4-argon', 'gemini-4-barium', 'gemini-3.8-flash'])
      assert.match(rows[0], /aria-label="复制 gemini-4-argon"/)
      assert.match(rows[1], /aria-label="复制 gemini-4-barium"/)
      assert.match(rows[1], /渠道基准/)
      assert.doesNotMatch(rows[1], /官方|价格待确认|NaN/)

      const priceClass = priceMode === 'group' ? 'group-price' : 'official-price'
      for (const row of rows.slice(0, 2)) {
        const amounts = [...row.matchAll(new RegExp(`<strong class="${priceClass}">([^<]+)<`, 'g'))]
          .map((match) => match[1])
        assert.deepEqual(amounts, priceMode === 'group'
          ? ['¥0.70', '¥3.50', '¥0.035', '¥4.20']
          : ['¥14.00', '¥70.00', '¥0.70', '¥84.00'])
      }
      if (priceMode === 'group') assert.match(html, /0\.35 倍率/)
    }
  } finally {
    await server.close()
  }
})

test('renders only supported models and their exact copy IDs in the 0.7 CC TEST group', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')
    const expectedIds = [
      'claude-opus-5-5', 'claude-opus-5', 'claude-opus-4-8', 'claude-opus-4-7',
      'claude-opus-4-6', 'claude-opus-4-5', 'claude-sonnet-5-5', 'claude-sonnet-5',
      'claude-sonnet-4-6', 'claude-sonnet-4-5', 'claude-haiku-4-5',
    ]

    for (const priceMode of ['group', 'official']) {
      const component = {
        ...pricing,
        setup(props, context) {
          const state = pricing.setup(props, context)
          state.activeCategory.value = 'anthropic'
          state.activeGroupId.value = 'anthropic-cc-test-0.7'
          state.priceMode.value = priceMode
          return state
        },
      }
      const html = await renderToString(createSSRApp(component))
      const rows = [...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
        .map(([row]) => row)
        .filter((row) => row.includes('class="model-id-cell"'))
      const ids = rows.map((row) => row.match(/<strong>(claude-[^<]+)<\/strong>/)[1])

      assert.deepEqual(ids, expectedIds, `${priceMode} should show only the 11 screenshot models`)
      for (const [index, row] of rows.entries()) {
        assert.ok(row.includes(`aria-label="复制 ${expectedIds[index]}"`))
        const priceClass = priceMode === 'group' ? 'group-price' : 'official-price'
        assert.equal([...row.matchAll(new RegExp(`class="${priceClass}"`, 'g'))].length, 4)
      }
      if (priceMode === 'group') {
        assert.match(html, /不保证长久，缓存 96–99%/)
        assert.match(html, /0\.7 倍率/)
      }
    }
  } finally {
    await server.close()
  }
})

test('renders Haiku 5.5 once with both context prices in the same row', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')
    const expectedGroupPrices = new Map([
      ['anthropic-max', ['¥0.14', '¥0.70', '¥0.70', '¥3.50', '¥0.014', '¥0.07', '¥0.84', '¥4.20']],
      ['anthropic-max-external', ['¥0.14', '¥0.70', '¥0.70', '¥3.50', '¥0.014', '¥0.07', '¥0.84', '¥4.20']],
    ])
    const expectedOfficialPrices = ['¥0.70', '¥3.50', '¥3.50', '¥17.50', '¥0.07', '¥0.35', '¥4.20', '¥21.00']

    for (const [groupId, expectedPrices] of expectedGroupPrices) {
      for (const priceMode of ['group', 'official']) {
        const component = {
          ...pricing,
          setup(props, context) {
            const state = pricing.setup(props, context)
            state.activeCategory.value = 'anthropic'
            state.activeGroupId.value = groupId
            state.priceMode.value = priceMode
            return state
          },
        }
        const html = await renderToString(createSSRApp(component))
        const rows = [...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
          .map(([row]) => row)
          .filter((row) => row.includes('<strong>claude-haiku-5-5</strong>'))

        assert.equal(rows.length, 1, `${groupId}/${priceMode} should show a single model row`)
        const [row] = rows
        assert.equal([...row.matchAll(/aria-label="复制 claude-haiku-5-5"/g)].length, 1)
        assert.match(row, /按单次请求的输入 tokens 数分档；输入、输出和缓存读取均按对应档计费/)
        const labels = [...row.matchAll(/<span class="context-tier-label">([^<]+)<\/span>/g)].map((match) => match[1])
        assert.deepEqual(labels, Array(4).fill(['输入 ≤ 10 万 tokens', '输入 &gt; 10 万 tokens']).flat())

        const priceClass = priceMode === 'group' ? 'group-price' : 'official-price'
        const amounts = [...row.matchAll(new RegExp(`<strong class="${priceClass}">([^<]+)<`, 'g'))]
          .map((match) => match[1])
        assert.deepEqual(amounts, priceMode === 'group' ? expectedPrices : expectedOfficialPrices)

        if (priceMode === 'official') {
          const dollars = [...row.matchAll(/<span class="official-usd">([^<]+)<\/span>/g)]
            .map((match) => match[1].trim())
          assert.deepEqual(dollars, ['$0.1', '$0.5', '$0.5', '$2.5', '$0.01', '$0.05', '$0.60', '$3.00'])
        }
      }
    }
  } finally {
    await server.close()
  }
})
