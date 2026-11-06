import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'

test('renders Argon in its own 0.5 group and keeps Barium in the 0.35 Gemini group', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')

    const cases = [
      {
        groupId: 'gemini-antigravity',
        count: 13,
        firstIds: ['gemini-4-barium', 'gemini-3.8-flash'],
        amounts: ['¥0.70', '¥3.50', '¥0.035', '¥4.20'],
        multiplier: /0\.35 倍率/,
      },
      {
        groupId: 'gemini-4-argon',
        count: 1,
        firstIds: ['gemini-4-argon'],
        amounts: ['¥1.00', '¥5.00', '¥0.05', '¥6.00'],
        multiplier: /0\.5 倍率/,
      },
    ]

    for (const { groupId, count, firstIds, amounts: groupAmounts, multiplier } of cases) {
      for (const priceMode of ['group', 'official']) {
        const component = {
          ...pricing,
          setup(props, context) {
            const state = pricing.setup(props, context)
            state.activeCategory.value = 'gemini'
            state.activeGroupId.value = groupId
            state.priceMode.value = priceMode
            return state
          },
        }
        const html = await renderToString(createSSRApp(component))
        const rows = [...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
          .map(([row]) => row)
          .filter((row) => row.includes('class="model-id-cell"'))
        const ids = rows.map((row) => row.match(/<strong>(gemini-[^<]+)<\/strong>/)[1])

        assert.equal(ids.length, count)
        assert.deepEqual(ids.slice(0, firstIds.length), firstIds)
        assert.ok(rows[0].includes(`aria-label="复制 ${firstIds[0]}"`))
        if (groupId === 'gemini-antigravity') {
          assert.ok(!ids.includes('gemini-4-argon'))
          assert.match(rows[0], /渠道基准/)
          assert.doesNotMatch(rows[0], /官方|价格待确认|NaN/)
        }

        const priceClass = priceMode === 'group' ? 'group-price' : 'official-price'
        const amounts = [...rows[0].matchAll(new RegExp(`<strong class="${priceClass}">([^<]+)<`, 'g'))]
          .map((match) => match[1])
        assert.deepEqual(amounts, priceMode === 'group'
          ? groupAmounts
          : ['¥14.00', '¥70.00', '¥0.70', '¥84.00'])
        if (priceMode === 'group') {
          const activeCard = html.match(/<button\b[^>]*class="[^"]*pricing-group-card[^"]*"[^>]*aria-pressed="true"[^>]*>[\s\S]*?<\/button>/)[0]
          assert.match(activeCard, multiplier)
        }
      }
    }
  } finally {
    await server.close()
  }
})

test('renders verified Qwen and MiMo IDs with RMB prices in their 0.45 groups', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')
    const cases = [
      {
        groupId: 'qwen',
        models: [
          { id: 'qwen3.8-max', official: ['¥12.00', '¥36.00', '¥1.50', '¥48.00'], group: ['¥5.40', '¥16.20', '¥0.68', '¥21.60'] },
          { id: 'qwen3.8-flash', official: ['¥0.80', '¥2.70', '¥0.10', '¥3.50'], group: ['¥0.36', '¥1.22', '¥0.045', '¥1.58'] },
        ],
      },
      {
        groupId: 'mimo',
        models: [
          { id: 'mimo-v2.6-pro', official: ['¥3.00', '¥6.00', '¥0.025', '¥9.00'], group: ['¥1.35', '¥2.70', '¥0.0113', '¥4.05'] },
          { id: 'mimo-v2.6-flash', official: ['¥1.00', '¥2.00', '¥0.02', '¥3.00'], group: ['¥0.45', '¥0.90', '¥0.009', '¥1.35'] },
        ],
      },
    ]

    for (const { groupId, models } of cases) {
      for (const priceMode of ['group', 'official']) {
        const component = {
          ...pricing,
          setup(props, context) {
            const state = pricing.setup(props, context)
            state.activeCategory.value = groupId
            state.activeGroupId.value = groupId
            state.priceMode.value = priceMode
            return state
          },
        }
        const html = await renderToString(createSSRApp(component))
        const rows = [...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
          .map(([row]) => row)
          .filter((row) => row.includes('class="model-id-cell"'))

        assert.equal(rows.length, models.length)
        for (const [index, model] of models.entries()) {
          const row = rows[index]
          assert.ok(row.includes(`<strong>${model.id}</strong>`))
          assert.ok(row.includes(`aria-label="复制 ${model.id}"`))
          const priceClass = priceMode === 'group' ? 'group-price' : 'official-price'
          const amounts = [...row.matchAll(new RegExp(`<strong class="${priceClass}">([^<]+)<`, 'g'))]
            .map((match) => match[1])
          assert.deepEqual(amounts, model[priceMode], `${groupId}/${priceMode}/${model.id}`)
          if (priceMode === 'group') assert.match(row, /省 55%/)
        }
        assert.doesNotMatch(html, /NaN|价格待确认|official-usd/)
        if (priceMode === 'group') {
          assert.match(html, /0\.45 倍率/)
          assert.match(html, /4\.5折/)
        }
      }
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
