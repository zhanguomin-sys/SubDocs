import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'

test('renders verified Gemini token prices and GPT-6.2 official references', async () => {
  const server = await createServer({
    configFile: false,
    plugins: [vue()],
    server: { middlewareMode: true },
    appType: 'custom',
  })

  try {
    const { default: pricing } = await server.ssrLoadModule('/docs/.vitepress/theme/components/ModelPricing.vue')

    const render = async (category, group, mode) => {
      const component = {
        ...pricing,
        setup(props, context) {
          const state = pricing.setup(props, context)
          state.activeCategory.value = category
          state.activeGroupId.value = group
          state.priceMode.value = mode
          return state
        },
      }
      const html = await renderToString(createSSRApp(component))
      const rows = new Map([...html.matchAll(/<tr\b[^>]*>[\s\S]*?<\/tr>/g)]
        .map(([row]) => row)
        .filter((row) => row.includes('class="model-id-cell"'))
        .map((row) => [row.match(/class="model-id-title"><strong>([^<]+)<\/strong>/)[1], row]))
      return { html, rows }
    }
    const groupGemini = await render('gemini', 'gemini-antigravity', 'group')
    const officialGemini = await render('gemini', 'gemini-antigravity', 'official')
    const amounts = (row, mode) => [...row.matchAll(new RegExp(`<strong class="${mode}-price">([^<]+)<`, 'g'))]
      .map((match) => match[1])

    assert.match(groupGemini.html, /0\.8 倍率/)
    assert.equal(groupGemini.rows.size, 17)
    assert.deepEqual(amounts(groupGemini.rows.get('gemini-4-neon'), 'group'), ['¥2.40', '¥12.00', '¥14.40'])
    assert.match(groupGemini.rows.get('gemini-4-neon'), /分组价待确认/)
    assert.deepEqual(amounts(groupGemini.rows.get('gemini-4-argon'), 'group'), ['¥1.60', '¥8.00', '¥0.08', '¥9.60'])
    assert.deepEqual(amounts(officialGemini.rows.get('gemini-4-neon'), 'official'), ['¥21.00', '¥105.00', '¥1.05', '¥126.00'])

    const groupGpt = await render('gpt', 'gpt-0.18', 'group')
    const officialGpt = await render('gpt', 'gpt-0.18', 'official')
    for (const id of ['gpt-6.2-astra', 'gpt-6.2-sol']) {
      assert.match(groupGpt.rows.get(id), /本站分组价格待确认/)
      assert.deepEqual(amounts(groupGpt.rows.get(id), 'group'), [])
    }
    assert.deepEqual(amounts(officialGpt.rows.get('gpt-6.2-astra'), 'official'), ['¥56.00', '¥280.00', '¥2.80', '¥336.00'])
    assert.deepEqual(amounts(officialGpt.rows.get('gpt-6.2-sol'), 'official'), ['¥14.00', '¥70.00', '¥0.70', '¥84.00'])
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
