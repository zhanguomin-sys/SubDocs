import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
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
  getSavingsPercent,
} from '../docs/.vitepress/theme/components/model-pricing-data.mjs'

const pricingComponentSource = readFileSync(
  new URL('../docs/.vitepress/theme/components/ModelPricing.vue', import.meta.url),
  'utf8',
)
const modelsDocSource = readFileSync(new URL('../docs/models.md', import.meta.url), 'utf8')
const vitepressConfigSource = readFileSync(new URL('../docs/.vitepress/config.mts', import.meta.url), 'utf8')

const isClose = (actual, expected) => Math.abs(actual - expected) < 1e-12

test('includes the updated GPT pricing groups in order', () => {
  assert.deepEqual(
    TEXT_GROUPS.map(({ id, name, multiplier }) => ({ id, name, multiplier })),
    [
      { id: 'pro-plus', name: 'GPT Plus 特惠分组（最近不稳定）', multiplier: 0.1 },
      { id: 'gpt-0.18', name: 'GPT Pro / Plus 混池分组', multiplier: 0.15 },
      { id: 'full', name: 'GPT 正价 Pro 满血分组', multiplier: 0.27 },
      { id: 'anthropic-main', name: '低价分组', multiplier: 0.2 },
      { id: 'anthropic-cc-test', name: 'Anthropic CC TEST 满分渠道', multiplier: 0.4 },
      { id: 'anthropic-cc-test-0.7', name: 'CC TEST 满分99缓（临时渠道，会下线）', multiplier: 0.7 },
      { id: 'anthropic-max', name: 'CC MAX 满血版本', multiplier: 1.4 },
      { id: 'anthropic-max-external', name: 'CC MAX 外接分组', multiplier: 1.4 },
      { id: 'grok-free', name: 'free号池', multiplier: 0.1 },
      { id: 'grok-4.5', name: 'heavy号池', multiplier: 0.3 },
      { id: 'gemini-antigravity', name: 'Gemini 分组（反重力 Antigravity 反代）', multiplier: 0.35 },
      { id: 'deepseek', name: 'DeepSeek 分组', multiplier: 0.45 },
      { id: 'domestic', name: '国产之光', multiplier: 0.45 },
      { id: 'kimi', name: 'Kimi 分组', multiplier: 0.45 },
      { id: 'hunyuan', name: '腾讯混元分组', multiplier: 0.45 },
    ],
  )
})

test('keeps the model category tabs in the requested order', () => {
  assert.deepEqual(
    MODEL_CATEGORIES.map(({ id, name, kind, groupIds, defaultGroupId }) => ({ id, name, kind, groupIds, defaultGroupId })),
    [
      { id: 'gpt', name: 'GPT', kind: 'text', groupIds: ['pro-plus', 'gpt-0.18', 'full'], defaultGroupId: 'gpt-0.18' },
      { id: 'anthropic', name: 'Anthropic', kind: 'text', groupIds: ['anthropic-main', 'anthropic-cc-test', 'anthropic-cc-test-0.7', 'anthropic-max', 'anthropic-max-external'], defaultGroupId: undefined },
      { id: 'grok', name: 'Grok', kind: 'text', groupIds: ['grok-free', 'grok-4.5'], defaultGroupId: undefined },
      { id: 'gemini', name: 'Gemini', kind: 'text', groupIds: ['gemini-antigravity'], defaultGroupId: undefined },
      { id: 'deepseek', name: 'DeepSeek', kind: 'text', groupIds: ['deepseek'], defaultGroupId: undefined },
      { id: 'domestic', name: 'GLM', kind: 'text', groupIds: ['domestic'], defaultGroupId: undefined },
      { id: 'kimi', name: 'Kimi', kind: 'text', groupIds: ['kimi'], defaultGroupId: undefined },
      { id: 'hunyuan', name: '腾讯混元', kind: 'text', groupIds: ['hunyuan'], defaultGroupId: undefined },
      { id: 'image', name: '生图', kind: 'image', groupIds: [], defaultGroupId: undefined },
    ],
  )
})

test('uses colored Claude Code, Grok and Gemini icons instead of letter placeholders', () => {
  const anthropic = MODEL_CATEGORIES.find((category) => category.id === 'anthropic')
  const grok = MODEL_CATEGORIES.find((category) => category.id === 'grok')
  const gemini = MODEL_CATEGORIES.find((category) => category.id === 'gemini')

  assert.equal(anthropic.mark, undefined)
  assert.match(anthropic.iconSvg, /fill="#D97757"/)
  assert.match(anthropic.iconSvg, /M4\.709 15\.955/)
  assert.match(grok.iconSvg, /fill="#141413"/)
  assert.equal(gemini.mark, undefined)
  assert.match(gemini.iconSvg, /fill="#8E75B2"/)
  assert.match(gemini.iconSvg, /M11\.04 19\.32/)
})

test('uses official local logos for DeepSeek, GLM and Kimi', () => {
  const deepseek = MODEL_CATEGORIES.find((category) => category.id === 'deepseek')
  const glm = MODEL_CATEGORIES.find((category) => category.id === 'domestic')
  const kimi = MODEL_CATEGORIES.find((category) => category.id === 'kimi')

  assert.deepEqual(
    [deepseek.iconSrc, glm.iconSrc, kimi.iconSrc],
    ['/brand-icons/deepseek.ico', '/brand-icons/glm.png', '/brand-icons/kimi.ico'],
  )

  for (const category of [deepseek, glm, kimi]) {
    assert.equal(category.mark, undefined)
    assert.equal(category.iconSvg, undefined)
  }
})

test('keeps category logos at icon size instead of inheriting VitePress document image styles', () => {
  assert.match(
    pricingComponentSource,
    /\.model-category-tabs \.model-category-icon\s*\{[\s\S]*?width: 25px;[\s\S]*?height: 25px;[\s\S]*?margin: 0;[\s\S]*?border: 0;[\s\S]*?box-shadow: none;[\s\S]*?cursor: inherit;/,
  )
})

test('uses the requested Anthropic group recommendations', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'anthropic-main')
  const ccTestGroup = TEXT_GROUPS.find((item) => item.id === 'anthropic-cc-test')

  assert.equal(group.name, '低价分组')
  assert.equal(group.description, '复杂架构设计推荐 Opus 5，日常任务推荐 Sonnet 5')
  assert.equal(ccTestGroup.description, '价格更低，适合 Claude Code 日常任务')
})

test('keeps the GPT Plus discount group at 0.1 with instability copy', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'pro-plus')

  assert.equal(group.name, 'GPT Plus 特惠分组（最近不稳定）')
  assert.equal(group.multiplier, 0.1)
  assert.match(group.description, /最近不稳定/)
})

test('keeps the GPT family models in the expected order', () => {
  for (const groupId of ['pro-plus', 'gpt-0.18', 'full']) {
    assert.deepEqual(
      getTextModelsForGroup(groupId).map((model) => model.id),
      [
        'gpt-6-astra',
        'gpt-6.1-sol',
        'gpt-6-sol',
        'gpt-5.6-sol',
        'gpt-5.6-terra',
        'gpt-5.6-luna',
        'gpt-5.5',
      ],
    )
  }
})

test('prices GPT-5.6 Terra from the $2 official input baseline in all three GPT groups', () => {
  const officialUsd = { input: 2, output: 15, cachedInput: 0.25 }
  const expectedGroupInputs = new Map([
    ['pro-plus', 0.2],
    ['gpt-0.18', 0.3],
    ['full', 0.54],
  ])

  for (const [groupId, expectedInput] of expectedGroupInputs) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const model = getTextModelsForGroup(groupId).find((item) => item.id === 'gpt-5.6-terra')

    assert.ok(model, `GPT-5.6 Terra should be available in ${groupId}`)
    assert.deepEqual(model.officialUsd, officialUsd)

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    assert.equal(price.official.input, 14)
    assert.ok(isClose(price.group.input, expectedInput))
  }
})

test('prices GPT-6.1 Sol from the official Standard baseline in all three GPT groups', () => {
  const officialUsd = { input: 2, output: 10, cachedInput: 0.1 }
  const expectedGroupPrices = new Map([
    ['pro-plus', { input: 0.2, output: 1, cachedInput: 0.01, total: 1.2 }],
    ['gpt-0.18', { input: 0.3, output: 1.5, cachedInput: 0.015, total: 1.8 }],
    ['full', { input: 0.54, output: 2.7, cachedInput: 0.027, total: 3.24 }],
  ])

  for (const [groupId, expectedGroup] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const model = getTextModelsForGroup(groupId).find((item) => item.id === 'gpt-6.1-sol')

    assert.ok(model, `GPT-6.1 Sol should be available in ${groupId}`)
    assert.deepEqual(model.officialUsd, officialUsd)

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    assert.equal(price.official.input, 14)
    assert.equal(price.official.output, 70)
    assert.ok(isClose(price.official.cachedInput, 0.7))
    assert.equal(price.official.total, 84)
    assert.ok(isClose(price.group.input, expectedGroup.input))
    assert.ok(isClose(price.group.output, expectedGroup.output))
    assert.ok(isClose(price.group.cachedInput, expectedGroup.cachedInput))
    assert.ok(isClose(price.group.total, expectedGroup.total))
  }
})

test('prices GPT-6 Sol from the official Standard baseline in all three GPT groups', () => {
  const officialUsd = { input: 2, output: 10, cachedInput: 0.2 }
  const expectedGroupPrices = new Map([
    ['pro-plus', { input: 0.2, output: 1, cachedInput: 0.02, total: 1.2 }],
    ['gpt-0.18', { input: 0.3, output: 1.5, cachedInput: 0.03, total: 1.8 }],
    ['full', { input: 0.54, output: 2.7, cachedInput: 0.054, total: 3.24 }],
  ])

  for (const [groupId, expectedGroup] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const model = getTextModelsForGroup(groupId).find((item) => item.id === 'gpt-6-sol')

    assert.ok(model, `GPT-6 Sol should be available in ${groupId}`)
    assert.deepEqual(model.officialUsd, officialUsd)

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    assert.equal(price.official.input, 14)
    assert.equal(price.official.output, 70)
    assert.ok(isClose(price.official.cachedInput, 1.4))
    assert.equal(price.official.total, 84)
    assert.ok(isClose(price.group.input, expectedGroup.input))
    assert.ok(isClose(price.group.output, expectedGroup.output))
    assert.ok(isClose(price.group.cachedInput, expectedGroup.cachedInput))
    assert.ok(isClose(price.group.total, expectedGroup.total))
  }
})

test('prices GPT-6 Astra in all three GPT groups with the revised baseline', () => {
  const proPlusModel = getTextModelsForGroup('pro-plus').find((model) => model.id === 'gpt-6-astra')
  const mixedModel = getTextModelsForGroup('gpt-0.18').find((model) => model.id === 'gpt-6-astra')
  const proModel = getTextModelsForGroup('full').find((model) => model.id === 'gpt-6-astra')

  assert.ok(proPlusModel)
  assert.ok(mixedModel)
  assert.ok(proModel)
  assert.deepEqual(proModel.officialUsd, {
    input: 10,
    output: 50,
    cachedInput: 1,
  })
  assert.deepEqual(proModel.billingOverridesUsd, { cachedInput: 2 })
  assert.equal(proPlusModel.unavailableMessage, '')
  assert.equal(mixedModel.unavailableMessage, '')
  assert.equal(proModel.unavailableMessage, '')

  const price = calculateTextPrice(proModel.officialUsd, 0.27, 'usd', proModel.billingOverridesUsd)
  assert.deepEqual(price.official, {
    input: 70,
    output: 350,
    cachedInput: 7,
    total: 420,
  })
  assert.ok(isClose(price.group.input, 2.7))
  assert.ok(isClose(price.group.output, 13.5))
  assert.ok(isClose(price.group.cachedInput, 0.54))
  assert.ok(isClose(price.group.total, 16.2))
})

test('orders all Anthropic groups from Fable 5.1 through Opus 5.5 to Opus 5', () => {
  const mainModels = getTextModelsForGroup('anthropic-main')
  const ccTestModels = getTextModelsForGroup('anthropic-cc-test')
  const ccMaxModels = getTextModelsForGroup('anthropic-max')
  const ccMaxExternalModels = getTextModelsForGroup('anthropic-max-external')
  const mainOrder = [
    'claude-opus-5-5',
    'claude-opus-5',
    'claude-opus-4-8',
    'claude-opus-4-7',
    'claude-opus-4-6',
    'claude-opus-4-5-20251101',
    'claude-sonnet-5-5',
    'claude-sonnet-5',
    'claude-sonnet-4-6',
    'claude-haiku-4-5-20251001',
    'claude-fable-5',
  ]
  const ccOrder = [
    ...mainOrder.filter((id) => !id.startsWith('claude-sonnet-')),
  ]
  const ccMaxOrder = [
    'claude-fable-5-1',
    ...mainOrder.slice(0, 9),
    'claude-sonnet-4-5-20250929',
    'claude-haiku-5-5',
    ...mainOrder.slice(9),
  ]

  assert.deepEqual(mainModels.map((model) => model.id), mainOrder)
  assert.deepEqual(ccTestModels.map((model) => model.id), ccOrder)
  assert.deepEqual(ccMaxModels.map((model) => model.id), ccMaxOrder)
  assert.deepEqual(ccMaxExternalModels.map((model) => model.id), ccMaxOrder)
  assert.equal(mainModels.at(-1).unavailableMessage, '此分组无 Fable 5')
  assert.equal(ccTestModels.at(-1).unavailableMessage, '此分组无 Fable 5')
  assert.equal(ccMaxModels.at(-1).unavailableMessage, '')
  assert.equal(ccMaxExternalModels.at(-1).unavailableMessage, '')
})

test('includes Claude Haiku 5.5 only in both CC MAX groups with both context prices', () => {
  const expectedGroupPrices = new Map([
    ['anthropic-max', [
      { input: 0.14, output: 0.7, cachedInput: 0.014, total: 0.84 },
      { input: 0.7, output: 3.5, cachedInput: 0.07, total: 4.2 },
    ]],
    ['anthropic-max-external', [
      { input: 0.14, output: 0.7, cachedInput: 0.014, total: 0.84 },
      { input: 0.7, output: 3.5, cachedInput: 0.07, total: 4.2 },
    ]],
  ])
  const expectedOfficialPrices = [
    { input: 0.7, output: 3.5, cachedInput: 0.07, total: 4.2 },
    { input: 3.5, output: 17.5, cachedInput: 0.35, total: 21 },
  ]

  for (const [groupId, expectedPrices] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const matches = getTextModelsForGroup(groupId).filter((item) => item.id === 'claude-haiku-5-5')

    assert.equal(matches.length, 1, `Claude Haiku 5.5 should appear once in ${groupId}`)
    const [model] = matches
    assert.equal(model.name, 'Claude Haiku 5.5')
    assert.equal(model.unavailableMessage, '')
    assert.deepEqual(model.officialUsd, { input: 0.1, output: 0.5, cachedInput: 0.01 })
    assert.deepEqual(model.officialLongContextUsd, { input: 0.5, output: 2.5, cachedInput: 0.05 })

    for (const [index, officialPrice] of [model.officialUsd, model.officialLongContextUsd].entries()) {
      const price = calculateTextPrice(officialPrice, group.multiplier)
      for (const field of ['input', 'output', 'cachedInput', 'total']) {
        assert.ok(isClose(price.official[field], expectedOfficialPrices[index][field]))
        assert.ok(isClose(price.group[field], expectedPrices[index][field]))
      }
    }

    assert.ok(getTextModelsForGroup(groupId).some((item) => item.id === 'claude-haiku-4-5-20251001'))
  }

  for (const group of TEXT_GROUPS.filter((item) => !expectedGroupPrices.has(item.id))) {
    assert.ok(!getTextModelsForGroup(group.id).some((item) => item.id === 'claude-haiku-5-5'))
  }
})

test('excludes every Sonnet model from the 0.4 CC TEST group', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'anthropic-cc-test')

  assert.equal(group.multiplier, 0.4)
  assert.ok(group.modelIds.every((id) => !id.startsWith('claude-sonnet-')))
})

test('prices only the 11 screenshot models in the temporary 0.7 CC TEST group', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'anthropic-cc-test-0.7')
  const models = getTextModelsForGroup(group.id)
  const expectedPrices = new Map([
    ['claude-opus-5-5', [2.8, 14, 0.14, 16.8]],
    ['claude-opus-5', [3.5, 17.5, 0.35, 21]],
    ['claude-opus-4-8', [3.5, 17.5, 0.35, 21]],
    ['claude-opus-4-7', [3.5, 17.5, 0.35, 21]],
    ['claude-opus-4-6', [3.5, 17.5, 0.35, 21]],
    ['claude-opus-4-5', [3.5, 17.5, 0.35, 21]],
    ['claude-sonnet-5-5', [1.4, 7, 0.07, 8.4]],
    ['claude-sonnet-5', [1.4, 7, 0.14, 8.4]],
    ['claude-sonnet-4-6', [2.1, 10.5, 0.21, 12.6]],
    ['claude-sonnet-4-5', [2.1, 10.5, 0.21, 12.6]],
    ['claude-haiku-4-5', [0.7, 3.5, 0.07, 4.2]],
  ])

  assert.equal(group.multiplier, 0.7)
  assert.match(group.name, /临时渠道，会下线/)
  assert.equal(group.description, '不保证长久，缓存 96–99%，其余与 CC TEST 0.4 倍分组一致')
  assert.deepEqual(models.map((model) => model.id), [...expectedPrices.keys()])

  for (const model of models) {
    assert.equal(model.unavailableMessage, '')
    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    const expected = expectedPrices.get(model.id)
    for (const [index, field] of ['input', 'output', 'cachedInput', 'total'].entries()) {
      assert.ok(isClose(price.group[field], expected[index]), `${model.id}/${field}`)
    }
  }
})

test('prices Claude Sonnet 5.5 in the low-price and both CC MAX groups', () => {
  const expectedGroupPrices = new Map([
    ['anthropic-main', { input: 0.4, output: 2, cachedInput: 0.02 }],
    ['anthropic-max', { input: 2.8, output: 14, cachedInput: 0.14 }],
    ['anthropic-max-external', { input: 2.8, output: 14, cachedInput: 0.14 }],
  ])

  for (const [groupId, expectedGroup] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const model = getTextModelsForGroup(groupId).find((item) => item.id === 'claude-sonnet-5-5')

    assert.ok(model, `Claude Sonnet 5.5 should be available in ${groupId}`)
    assert.deepEqual(model.officialUsd, { input: 2, output: 10, cachedInput: 0.1 })
    assert.equal(model.unavailableMessage, '')

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    assert.equal(price.official.input, 14)
    assert.equal(price.official.output, 70)
    assert.ok(isClose(price.official.cachedInput, 0.7))
    assert.ok(isClose(price.group.input, expectedGroup.input))
    assert.ok(isClose(price.group.output, expectedGroup.output))
    assert.ok(isClose(price.group.cachedInput, expectedGroup.cachedInput))
  }
})

test('prices Claude Sonnet 5 at the reduced official baseline in all supported groups', () => {
  const expectedGroupPrices = new Map([
    ['anthropic-main', { input: 0.4, output: 2, cachedInput: 0.04, total: 2.4 }],
    ['anthropic-max', { input: 2.8, output: 14, cachedInput: 0.28, total: 16.8 }],
    ['anthropic-max-external', { input: 2.8, output: 14, cachedInput: 0.28, total: 16.8 }],
  ])
  const expectedOfficial = { input: 14, output: 70, cachedInput: 1.4, total: 84 }

  for (const [groupId, expectedGroup] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const matches = getTextModelsForGroup(groupId).filter((item) => item.id === 'claude-sonnet-5')

    assert.equal(matches.length, 1, `Claude Sonnet 5 should appear once in ${groupId}`)
    const [model] = matches
    assert.deepEqual(model.officialUsd, { input: 2, output: 10, cachedInput: 0.2 })
    assert.equal(model.unavailableMessage, '')

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    for (const field of ['input', 'output', 'cachedInput', 'total']) {
      assert.ok(isClose(price.official[field], expectedOfficial[field]))
      assert.ok(isClose(price.group[field], expectedGroup[field]))
    }
  }
})

test('includes Claude Opus 5.5 once at official pricing in every Anthropic group', () => {
  const expectedGroupPrices = new Map([
    ['anthropic-main', { input: 0.8, output: 4, cachedInput: 0.04, total: 4.8 }],
    ['anthropic-cc-test', { input: 1.6, output: 8, cachedInput: 0.08, total: 9.6 }],
    ['anthropic-cc-test-0.7', { input: 2.8, output: 14, cachedInput: 0.14, total: 16.8 }],
    ['anthropic-max', { input: 5.6, output: 28, cachedInput: 0.28, total: 33.6 }],
    ['anthropic-max-external', { input: 5.6, output: 28, cachedInput: 0.28, total: 33.6 }],
  ])
  const category = MODEL_CATEGORIES.find((item) => item.id === 'anthropic')
  assert.deepEqual(category.groupIds, [...expectedGroupPrices.keys()])

  for (const [groupId, expectedGroup] of expectedGroupPrices) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const matches = getTextModelsForGroup(groupId).filter((item) => item.id === 'claude-opus-5-5')

    assert.equal(matches.length, 1, `Claude Opus 5.5 should appear once in ${groupId}`)
    const [model] = matches
    assert.deepEqual(model.officialUsd, { input: 4, output: 20, cachedInput: 0.2 })
    assert.equal(model.unavailableMessage, '')

    const price = calculateTextPrice(model.officialUsd, group.multiplier)
    assert.equal(price.official.input, 28)
    assert.equal(price.official.output, 140)
    assert.ok(isClose(price.official.cachedInput, 1.4))
    assert.equal(price.official.total, 168)
    assert.ok(isClose(price.group.input, expectedGroup.input))
    assert.ok(isClose(price.group.output, expectedGroup.output))
    assert.ok(isClose(price.group.cachedInput, expectedGroup.cachedInput))
    assert.ok(isClose(price.group.total, expectedGroup.total))
  }
})

test('uses Anthropic official pricing for Claude Fable 5.1 in the CC MAX groups', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'anthropic-max')
  const externalGroup = TEXT_GROUPS.find((item) => item.id === 'anthropic-max-external')
  const model = getTextModelsForGroup(group.id).find((item) => item.id === 'claude-fable-5-1')

  assert.deepEqual(model.officialUsd, { input: 10, output: 50, cachedInput: 0.25 })
  assert.equal(group.multiplier, 1.4)
  assert.equal(externalGroup.multiplier, 1.4)

  const price = calculateTextPrice(model.officialUsd, externalGroup.multiplier)
  assert.deepEqual(price.official, { input: 70, output: 350, cachedInput: 1.75, total: 420 })
  assert.ok(isClose(price.group.input, 14))
  assert.ok(isClose(price.group.output, 70))
  assert.ok(isClose(price.group.cachedInput, 0.35))
  assert.ok(isClose(price.group.total, 84))
})

test('replaces unavailable group prices with one row message', () => {
  assert.match(pricingComponentSource, /priceMode === 'group' && model\.unavailableMessage/)
  assert.match(pricingComponentSource, /class="model-unavailable-cell"/)
  assert.match(pricingComponentSource, /\{\{ model\.unavailableMessage \}\}/)
})

test('shows the free and heavy Grok pools with the same models and requested multipliers', () => {
  const freeGroup = TEXT_GROUPS.find((item) => item.id === 'grok-free')
  const heavyGroup = TEXT_GROUPS.find((item) => item.id === 'grok-4.5')
  const expectedModels = [
    {
      id: 'grok-4.7',
      description: 'xAI 新一代模型，适合复杂代码、智能体任务和实时搜索',
      officialUsd: { input: 2, output: 6, cachedInput: 0.5 },
    },
    {
      id: 'grok-4.6',
      description: 'XAI最新模型，性能直逼OPUS 5跟GPT 5.6 SOL，限制低，速度快，当前非常火热。',
      officialUsd: { input: 2, output: 6, cachedInput: 0.5 },
    },
    {
      id: 'grok-4.5',
      description: '速度比 GPT 日常分组更快，风控相对低，支持实时搜索和工具调用',
      officialUsd: { input: 2, output: 6, cachedInput: 0.2 },
    },
  ]

  assert.equal(freeGroup.multiplier, 0.1)
  assert.equal(heavyGroup.multiplier, 0.3)
  assert.deepEqual(
    getTextModelsForGroup('grok-free').map(({ id, description, officialUsd }) => ({ id, description, officialUsd })),
    expectedModels,
  )
  assert.deepEqual(
    getTextModelsForGroup('grok-4.5').map(({ id, description, officialUsd }) => ({ id, description, officialUsd })),
    expectedModels,
  )
})

test('shows Gemini 4 Argon and Barium first in the Gemini Antigravity group', () => {
  assert.deepEqual(
    getTextModelsForGroup('gemini-antigravity').map((model) => model.id),
    [
      'gemini-4-argon',
      'gemini-4-barium',
      'gemini-3.8-flash',
      'gemini-3.7-flash',
      'gemini-3.6-flash-tiered',
      'gemini-3.6-flash',
      'gemini-3.5-flash',
      'gemini-3.1-pro-preview',
      'gemini-3.1-flash-lite',
      'gemini-3.1-flash-lite-preview',
      'gemini-3-pro-preview',
      'gemini-3-flash',
      'gemini-3-flash-preview',
      'gemini-2.5-pro',
      'gemini-2.5-flash',
      'gemini-2.5-flash-lite',
    ],
  )
})

test('shows GLM-5.3-Flash, GLM-5.3, GLM-5.2 and LongCat-2.0 together in the RMB domestic group', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'domestic')

  assert.equal(group.name, '国产之光')
  assert.equal(group.multiplier, 0.45)
  assert.equal(group.currency, 'cny')
  assert.match(group.description, /逆向/)
  assert.deepEqual(
    getTextModelsForGroup('domestic').map(({ id, officialCny }) => ({ id, officialCny })),
    [
      { id: 'glm-5.3-flash', officialCny: { input: 0.8, output: 2.8, cachedInput: 0.23 } },
      { id: 'glm-5.3', officialCny: { input: 8, output: 28, cachedInput: 2 } },
      { id: 'glm-5.2', officialCny: { input: 8, output: 28, cachedInput: 2 } },
      { id: 'LongCat-2.0', officialCny: { input: 2, output: 8, cachedInput: 0.04 } },
    ],
  )
})

test('calculates GLM-5.3-Flash from the official RMB baseline at 0.45x', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'domestic')
  const model = getTextModelsForGroup('domestic').find((item) => item.id === 'glm-5.3-flash')

  assert.ok(model, 'GLM-5.3-Flash should be available in the domestic group')
  const price = calculateTextPrice(model.officialCny, group.multiplier, group.currency)

  assert.equal(price.official.input, 0.8)
  assert.equal(price.official.output, 2.8)
  assert.equal(price.official.cachedInput, 0.23)
  assert.ok(isClose(price.official.total, 3.6))
  assert.ok(isClose(price.group.input, 0.36))
  assert.ok(isClose(price.group.output, 1.26))
  assert.ok(isClose(price.group.cachedInput, 0.1035))
  assert.ok(isClose(price.group.total, 1.62))
})

test('shows only Kimi K3 at the verified RMB price with a 0.45 multiplier', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'kimi')
  const models = getTextModelsForGroup('kimi')

  assert.equal(group.name, 'Kimi 分组')
  assert.equal(group.multiplier, 0.45)
  assert.equal(group.currency, 'cny')
  assert.deepEqual(
    models.map(({ id, name, officialCny }) => ({ id, name, officialCny })),
    [
      {
        id: 'kimi-k3',
        name: 'Kimi K3',
        officialCny: { input: 20, output: 100, cachedInput: 2 },
      },
    ],
  )

  const k3Price = calculateTextPrice(models[0].officialCny, group.multiplier, group.currency)
  assert.deepEqual(k3Price.official, { input: 20, output: 100, cachedInput: 2, total: 120 })
  assert.ok(isClose(k3Price.group.input, 9))
  assert.ok(isClose(k3Price.group.output, 45))
  assert.ok(isClose(k3Price.group.cachedInput, 0.9))
  assert.ok(isClose(k3Price.group.total, 54))
  assert.equal(getEquivalentDiscount(group.multiplier, group.currency), '4.5折')
  assert.equal(getSavingsPercent(group.multiplier, group.currency), 55)
})

test('calculates hy4-preview from the official RMB baseline at a 0.45 multiplier', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'hunyuan')
  const models = getTextModelsForGroup('hunyuan')

  assert.equal(group.name, '腾讯混元分组')
  assert.equal(group.multiplier, 0.45)
  assert.equal(group.currency, 'cny')
  assert.deepEqual(
    models.map(({ id, name, officialCny }) => ({ id, name, officialCny })),
    [
      {
        id: 'hy4-preview',
        name: 'hy4-preview',
        officialCny: { input: 6, output: 18, cachedInput: 0.3 },
      },
    ],
  )

  const price = calculateTextPrice(models[0].officialCny, group.multiplier, group.currency)
  assert.deepEqual(price.official, { input: 6, output: 18, cachedInput: 0.3, total: 24 })
  assert.ok(isClose(price.group.input, 2.7))
  assert.ok(isClose(price.group.output, 8.1))
  assert.ok(isClose(price.group.cachedInput, 0.135))
  assert.ok(isClose(price.group.total, 10.8))
  assert.equal(getEquivalentDiscount(group.multiplier, group.currency), '4.5折')
  assert.equal(getSavingsPercent(group.multiplier, group.currency), 55)
})

test('shows DeepSeek V4.1 Flash first, followed by V4 Flash and Pro, at a 0.45 multiplier', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'deepseek')
  const models = getTextModelsForGroup(group.id)

  assert.equal(group.name, 'DeepSeek 分组')
  assert.equal(group.multiplier, 0.45)
  assert.equal(group.currency, 'cny')
  assert.match(group.description, /周一至周五/)
  assert.deepEqual(
    models.map(({ id, name, officialCny }) => ({ id, name, officialCny })),
    [
      {
        id: 'deepseek-flash',
        name: 'DeepSeek V4.1 Flash',
        officialCny: { input: 1, output: 4, cachedInput: 0.02 },
      },
      {
        id: 'deepseek-v4-flash',
        name: 'DeepSeek V4 Flash 0731',
        officialCny: { input: 1, output: 4, cachedInput: 0.02 },
      },
      {
        id: 'deepseek-v4-pro',
        name: 'DeepSeek V4 Pro 0813',
        officialCny: { input: 4.5, output: 13.5, cachedInput: 0.15 },
      },
    ],
  )

  const price = calculateTextPrice(models[0].officialCny, group.multiplier, group.currency)
  assert.deepEqual(price.official, { input: 1, output: 4, cachedInput: 0.02, total: 5 })
  assert.ok(isClose(price.group.input, 0.45))
  assert.ok(isClose(price.group.output, 1.8))
  assert.ok(isClose(price.group.cachedInput, 0.009))
  assert.ok(isClose(price.group.total, 2.25))

  assert.deepEqual(models[0].officialPeakCny, { input: 2, output: 8, cachedInput: 0.04 })
  const peakPrice = calculateTextPrice(models[0].officialPeakCny, group.multiplier, group.currency)
  assert.ok(isClose(peakPrice.group.input, 0.9))
  assert.ok(isClose(peakPrice.group.output, 3.6))
  assert.ok(isClose(peakPrice.group.cachedInput, 0.018))
  assert.ok(isClose(peakPrice.group.total, 4.5))
})

test('calculates DeepSeek V4 Pro peak and off-peak RMB prices at a 0.45 multiplier', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'deepseek')
  const model = getTextModelsForGroup(group.id).find((item) => item.id === 'deepseek-v4-pro')

  assert.equal(group.multiplier, 0.45)
  assert.equal(group.currency, 'cny')
  const price = calculateTextPrice(model.officialCny, group.multiplier, group.currency)
  assert.deepEqual(price.official, { input: 4.5, output: 13.5, cachedInput: 0.15, total: 18 })
  assert.ok(isClose(price.group.input, 2.025))
  assert.ok(isClose(price.group.output, 6.075))
  assert.ok(isClose(price.group.cachedInput, 0.0675))
  assert.ok(isClose(price.group.total, 8.1))

  assert.deepEqual(model.officialPeakCny, { input: 9, output: 27, cachedInput: 0.3 })
  const peakPrice = calculateTextPrice(model.officialPeakCny, group.multiplier, group.currency)
  assert.ok(isClose(peakPrice.group.input, 4.05))
  assert.ok(isClose(peakPrice.group.output, 12.15))
  assert.ok(isClose(peakPrice.group.cachedInput, 0.135))
  assert.ok(isClose(peakPrice.group.total, 16.2))
})

test('shows DeepSeek peak and off-peak rows with clear Beijing busy hours', () => {
  assert.match(pricingComponentSource, /<th v-if="isDeepSeekCategory" scope="col">计费时段<\/th>/)
  assert.match(pricingComponentSource, /高峰时段为北京时间周一至周五 09:00-12:00、14:00-18:00，其余为空闲时段/)
  assert.match(pricingComponentSource, /officialPeakCny/)
  assert.doesNotMatch(pricingComponentSource, /暂按闲时基准全天收费/)
})

test('explains the Astra billing exception and removes the generic pricing footer', () => {
  assert.doesNotMatch(modelsDocSource, /DeepSeek V4 峰谷价格/)
  assert.doesNotMatch(modelsDocSource, /model not found|403/)
  assert.doesNotMatch(modelsDocSource, /我的 API Key 支持哪些模型|切换模型需要重新配置吗/)
  assert.match(pricingComponentSource, /GPT-6 Astra 计费说明/)
  assert.match(pricingComponentSource, /输入和输出按官方公开价 \$10、\$50 计算/)
  assert.match(pricingComponentSource, /社区长期观察并经大量用户实际调用验证/)
  assert.match(pricingComponentSource, /实际按 \$2 计费/)
  assert.match(pricingComponentSource, /Astra 缓存读取因此按实测基准 \$2 乘对应分组倍率计算/)
  assert.match(pricingComponentSource, /官方价格或计费规则发生变化后，本站会尽快同步调整/)
  assert.doesNotMatch(pricingComponentSource, /文本类模型官方价格按当前公开标准价和固定汇率换算/)
  assert.doesNotMatch(pricingComponentSource, /页面价格用于说明和对比/)
  assert.match(modelsDocSource, /价格表第一列是模型 ID，点击旁边按钮复制/)
  assert.match(modelsDocSource, /只修改客户端里的 `Model`；Base URL 和 API Key 保持不变/)
})

test('uses the displayed official price baselines for GPT-5.6', () => {
  assert.deepEqual(
    getTextModelsForGroup('gpt-0.18')
      .filter((model) => model.id.startsWith('gpt-5.6-'))
      .map((model) => model.officialUsd),
    [
      { input: 5, output: 30, cachedInput: 0.5 },
      { input: 2, output: 15, cachedInput: 0.25 },
      { input: 1, output: 6, cachedInput: 0.1 },
    ],
  )
})

test('keeps verified Gemini price baselines including the published Barium channel rate', () => {
  assert.deepEqual(
    getTextModelsForGroup('gemini-antigravity').map(({ id, officialUsd }) => ({ id, officialUsd })),
    [
      { id: 'gemini-4-argon', officialUsd: { input: 2, output: 10, cachedInput: 0.1 } },
      { id: 'gemini-4-barium', officialUsd: { input: 2, output: 10, cachedInput: 0.1 } },
      { id: 'gemini-3.8-flash', officialUsd: { input: 0.75, output: 3.75, cachedInput: 0.075 } },
      { id: 'gemini-3.7-flash', officialUsd: { input: 0.75, output: 3.75, cachedInput: 0.075 } },
      { id: 'gemini-3.6-flash-tiered', officialUsd: { input: 0.75, output: 3.75, cachedInput: 0.075 } },
      { id: 'gemini-3.6-flash', officialUsd: { input: 0.75, output: 3.75, cachedInput: 0.075 } },
      { id: 'gemini-3.5-flash', officialUsd: { input: 1.5, output: 9, cachedInput: 0.15 } },
      { id: 'gemini-3.1-pro-preview', officialUsd: { input: 2, output: 12, cachedInput: 0.2 } },
      { id: 'gemini-3.1-flash-lite', officialUsd: { input: 0.25, output: 1.5, cachedInput: 0.025 } },
      { id: 'gemini-3.1-flash-lite-preview', officialUsd: { input: 0.25, output: 1.5, cachedInput: 0.025 } },
      { id: 'gemini-3-pro-preview', officialUsd: { input: 2, output: 12, cachedInput: 0.2 } },
      { id: 'gemini-3-flash', officialUsd: { input: 0.5, output: 3, cachedInput: 0.05 } },
      { id: 'gemini-3-flash-preview', officialUsd: { input: 0.5, output: 3, cachedInput: 0.05 } },
      { id: 'gemini-2.5-pro', officialUsd: { input: 1.25, output: 10, cachedInput: 0.125 } },
      { id: 'gemini-2.5-flash', officialUsd: { input: 0.3, output: 2.5, cachedInput: 0.03 } },
      { id: 'gemini-2.5-flash-lite', officialUsd: { input: 0.1, output: 0.4, cachedInput: 0.01 } },
    ],
  )
})

test('calculates Gemini 3.8 Flash with a 0.35 group multiplier', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'gemini-antigravity')
  const model = getTextModelsForGroup(group.id).find((item) => item.id === 'gemini-3.8-flash')

  assert.equal(group.multiplier, 0.35)
  assert.ok(model, 'Gemini 3.8 Flash should be available in the Gemini group')
  const price = calculateTextPrice(model.officialUsd, group.multiplier)
  assert.ok(isClose(price.group.input, 0.2625))
  assert.ok(isClose(price.group.output, 1.3125))
  assert.ok(isClose(price.group.cachedInput, 0.02625))
  assert.ok(isClose(price.group.total, 1.575))
})

test('keeps Gemini descriptions customer-facing with dates or concrete use cases', () => {
  const models = new Map(getTextModelsForGroup('gemini-antigravity').map((model) => [model.id, model]))
  const forbidden = /旧预览|预览 ID|上一代|后台|Moxing|没的写/

  for (const model of models.values()) {
    assert.doesNotMatch(model.description, forbidden, model.id)
  }

  assert.ok(models.has('gemini-3.8-flash'))
  assert.match(models.get('gemini-3.8-flash').description, /2026-09/)
  assert.match(models.get('gemini-3.8-flash').description, /2026-12-31/)
  assert.match(models.get('gemini-3.7-flash').description, /2026-08/)
  assert.match(models.get('gemini-3.7-flash').description, /2026-12-31/)
  assert.match(models.get('gemini-3.6-flash').description, /2026-07/)
  assert.match(models.get('gemini-3.5-flash').description, /2026-05/)
  assert.match(models.get('gemini-3.1-pro-preview').description, /2026-02/)
  assert.match(models.get('gemini-3-pro-preview').description, /2025-11/)
  assert.match(models.get('gemini-2.5-pro').description, /2025-06/)
  assert.match(models.get('gemini-2.5-flash-lite').description, /高频|低延迟/)
})

test('highlights only Gemini 3.8 Flash as the featured model', () => {
  const models = getTextModelsForGroup('gemini-antigravity')
  const featuredModels = models.filter((item) => item.featured)
  const model = featuredModels[0]

  assert.equal(featuredModels.length, 1)
  assert.equal(model.id, 'gemini-3.8-flash')
  assert.equal(model.featured, true)
  assert.equal(model.featuredLabel, '主推')
  assert.match(pricingComponentSource, /'is-featured-model': model\.featured/)
  assert.match(pricingComponentSource, /class="featured-model-badge"/)
})

test('converts official USD prices to RMB at the fixed exchange rate', () => {
  const price = calculateTextPrice(
    { input: 5, output: 30, cachedInput: 0.5 },
    0.1,
  )

  assert.deepEqual(price.official, {
    input: 35,
    output: 210,
    cachedInput: 3.5,
    total: 245,
  })
})

test('uses the group multiplier directly on the official USD number', () => {
  const price = calculateTextPrice(
    { input: 5, output: 30, cachedInput: 0.5 },
    0.25,
  )

  assert.ok(isClose(price.group.input, 1.25))
  assert.ok(isClose(price.group.output, 7.5))
  assert.ok(isClose(price.group.cachedInput, 0.125))
  assert.ok(isClose(price.group.total, 8.75))
})

test('calculates GLM group prices directly in RMB without USD conversion', () => {
  const group = TEXT_GROUPS.find((item) => item.id === 'domestic')
  const price = calculateTextPrice(
    { input: 8, output: 28, cachedInput: 2 },
    group.multiplier,
    group.currency,
  )

  assert.deepEqual(price.official, {
    input: 8,
    output: 28,
    cachedInput: 2,
    total: 36,
  })
  assert.ok(isClose(price.group.input, 3.6))
  assert.ok(isClose(price.group.output, 12.6))
  assert.ok(isClose(price.group.cachedInput, 0.9))
  assert.ok(isClose(price.group.total, 16.2))
})

test('expresses the active multipliers as rounded equivalent discounts', () => {
  assert.equal(getEquivalentDiscount(0.1), '0.1折')
  assert.equal(getEquivalentDiscount(0.15), '0.2折')
  assert.equal(getEquivalentDiscount(0.25), '0.4折')
  assert.equal(getEquivalentDiscount(0.3), '0.4折')
  assert.equal(getEquivalentDiscount(1.9), '2.7折')
  assert.equal(getEquivalentDiscount(0.45, 'cny'), '4.5折')
  assert.equal(getSavingsPercent(0.45, 'cny'), 55)
})

test('calculates the revised group totals from the official USD baseline', () => {
  const officialUsd = { input: 5, output: 30, cachedInput: 0.5 }
  const expectedTotals = new Map([
    ['pro-plus', 3.5],
    ['gpt-0.18', 5.25],
    ['full', 9.45],
    ['anthropic-main', 7],
    ['anthropic-cc-test', 14],
    ['anthropic-max', 49],
    ['anthropic-max-external', 49],
    ['grok-free', 3.5],
    ['grok-4.5', 10.5],
    ['gemini-antigravity', 12.25],
  ])

  for (const [groupId, expectedTotal] of expectedTotals) {
    const group = TEXT_GROUPS.find((item) => item.id === groupId)
    const price = calculateTextPrice(officialUsd, group.multiplier)

    assert.ok(isClose(price.group.total, expectedTotal), groupId)
  }
})

test('keeps the model pricing page out of the global navigation chrome', () => {
  assert.match(modelsDocSource, /^navbar:\s*false$/m)
  assert.match(modelsDocSource, /^sidebar:\s*false$/m)
  assert.doesNotMatch(vitepressConfigSource, /text:\s*'模型价格',\s*link:\s*'\/models'/)
  assert.match(vitepressConfigSource, /excludeByGlobPattern:\s*\[[^\]]*'models\.md'/s)
})

test('reserves the pricing-group area while official prices are selected', () => {
  assert.match(pricingComponentSource, /v-else\s+class="pricing-groups pricing-groups--placeholder"/)
  assert.match(pricingComponentSource, /v-for="group in activeGroups"/)
  assert.match(pricingComponentSource, /class="pricing-group-card pricing-group-placeholder"/)
})

test('formats RMB amounts without noisy trailing zeroes', () => {
  assert.equal(formatCny(245), '¥245.00')
  assert.equal(formatCny(0.525), '¥0.53')
  assert.equal(formatCny(1.25), '¥1.25')
})

test('formats tiny non-zero RMB amounts without rounding them to zero', () => {
  assert.equal(formatCny(0.0049), '¥0.0049')
  assert.equal(formatCny(0.025), '¥0.025')
})

test('keeps the requested drawing prices in RMB without applying USD conversion', () => {
  assert.equal(EXCHANGE_RATE, 7)
  assert.match(IMAGE_GROUP.description, /人民币/)
  assert.equal(calculateImagePriceCny(0.03), 0.03)
  assert.equal(calculateImagePriceCny(0.05), 0.05)
  assert.equal(calculateImagePriceCny(0.2), 0.2)
  const requestedPrices = new Map(
    IMAGE_MODELS.map((model) => [model.id, calculateImagePriceCny(model.groupCnyPerImage)]),
  )

  assert.deepEqual(
    [...requestedPrices].filter(([id]) => ['gpt-image-1k-th', 'gpt-image-2', 'gpt-image-2-adobe'].includes(id)),
    [
      ['gpt-image-1k-th', 0.03],
      ['gpt-image-2', 0.05],
      ['gpt-image-2-adobe', 0.12],
    ],
  )

  const gptImage2 = IMAGE_MODELS.find((model) => model.id === 'gpt-image-2')
  assert.deepEqual(gptImage2.sizePricesCny, [
    { size: '1K', cnyPerImage: 0.04 },
    { size: '2K', cnyPerImage: 0.05 },
    { size: '4K', cnyPerImage: 0.08 },
  ])
  assert.equal(gptImage2.spec, '1K / 2K / 4K')
  assert.match(pricingComponentSource, /v-if="model\.sizePricesCny"/)
  assert.match(pricingComponentSource, /按分辨率计费/)
})

test('places the latest recommended GPT Image 2.5 models first at GPT Image 2 prices', () => {
  const gptImage2 = IMAGE_MODELS.find((model) => model.id === 'gpt-image-2')
  const latestModels = IMAGE_MODELS.slice(0, 2)

  assert.deepEqual(
    latestModels.map(({ id, name, description, recommended }) => ({ id, name, description, recommended })),
    [
      {
        id: 'gpt-image-2.5-flare',
        name: 'GPT Image 2.5 Flare',
        description: '最新推荐模型，适合快速生成、普通生图和批量任务',
        recommended: true,
      },
      {
        id: 'gpt-image-2.5-sunburst',
        name: 'GPT Image 2.5 Sunburst',
        description: '最新推荐模型，适合高清、精修和重要成品',
        recommended: true,
      },
    ],
  )

  for (const model of latestModels) {
    assert.equal(model.route, '/v1/images/generations')
    assert.equal(model.spec, gptImage2.spec)
    assert.equal(model.groupCnyPerImage, gptImage2.groupCnyPerImage)
    assert.deepEqual(model.sizePricesCny, gptImage2.sizePricesCny)
  }

  assert.equal(gptImage2.recommended, undefined)
  assert.deepEqual(
    IMAGE_MODELS.filter((model) => model.recommended).map((model) => model.id),
    ['gpt-image-2.5-flare', 'gpt-image-2.5-sunburst'],
  )
  assert.match(pricingComponentSource, /v-if="model\.recommended"/)
  assert.match(pricingComponentSource, /<span v-if="model\.recommended">最新推荐模型<\/span>/)
  assert.match(pricingComponentSource, /IMAGE_MODELS\.find\(\(model\) => model\.id === 'gpt-image-2\.5-flare'\)/)
})

test('shows the unified size-price note only for gpt-image-2-adobe', () => {
  const adobe = IMAGE_MODELS.find((model) => model.id === 'gpt-image-2-adobe')

  assert.equal(adobe.priceNote, '1/2/4k统一价')
  assert.match(pricingComponentSource, /model\.priceNote \?\? '当前分组默认价'/)
})

test('removes nonexistent GPT Image size aliases and the YS model from the pricing page', () => {
  const modelIds = new Set(IMAGE_MODELS.map((model) => model.id))

  for (const removedId of [
    'gpt-image-2-4k',
    'gpt-image-1k-adobe',
    'gpt-image-2k-adobe',
    'gpt-image-4k-adobe',
    'gpt-image-4k-ys',
  ]) {
    assert.equal(modelIds.has(removedId), false, removedId)
  }
})

test('publishes only the two requested Nano Banana models and RMB prices', () => {
  assert.deepEqual(
    IMAGE_MODELS
      .filter((model) => model.id.startsWith('nano-banana'))
      .map(({ id, groupCnyPerImage }) => ({ id, groupCnyPerImage })),
    [
      { id: 'nano-banana-pro', groupCnyPerImage: 0.25 },
      { id: 'nano-banana-2', groupCnyPerImage: 0.12 },
    ],
  )
})

test('publishes Grok Imagine Image at 0.10 RMB per image', () => {
  const grok = IMAGE_MODELS.find((model) => model.id === 'grok-imagine-image')

  assert.ok(grok, 'grok-imagine-image should be published')
  assert.deepEqual(
    {
      id: grok.id,
      name: grok.name,
      route: grok.route,
      spec: grok.spec,
      groupCnyPerImage: grok.groupCnyPerImage,
    },
    {
      id: 'grok-imagine-image',
      name: 'Grok Imagine Image',
      route: '/v1/images/generations',
      spec: '自动尺寸',
      groupCnyPerImage: 0.10,
    },
  )
})

test('keeps image model descriptions customer-facing without backend mapping wording', () => {
  const forbidden = /Google|Gemini|Firefly|Partner Model|账号渠道|YS 渠道|xAI|后台|映射/

  for (const model of IMAGE_MODELS) {
    assert.doesNotMatch(model.description, forbidden, model.id)
    assert.doesNotMatch(model.spec, /渠道档/, model.id)
  }
})
