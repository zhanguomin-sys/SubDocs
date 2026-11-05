'use strict';

const $ = (id) => document.getElementById(id);
const keyInput = $('key');
const connectButton = $('connect');
const restoreButton = $('restore');
const removeButton = $('remove-models');
const shell = $('shell');

// One panel, four tools. Codex and Claude are the two main ones, picked by
// the tabs at the top; Codex is the default. The image tool and the domestic
// model import are reached from small entries lower down. All of them reuse
// the same card, field and button.
const TOOLS = {
  codex: {
    label: 'Codex 状态',
    button: '一键连接 Codex',
    placeholder: '粘贴 API Key',
    restoreTip: '取消中转站连接：Codex 改回用 OpenAI 官方账号登录，并关闭正在运行的 Codex。',
  },
  image: {
    label: '生图工具',
    button: '一键导入生图工具',
    placeholder: '粘贴画图分组的 API Key',
    tag: '现在导入的是：生图工具的 Key',
    guide: '到网站「API 密钥」页复制画图分组的密钥，粘贴到上面。导入完成前不要关闭本窗口。',
  },
  claude: {
    label: 'Claude 桌面版',
    button: '一键连接 Claude',
    placeholder: '粘贴 API Key，并选择 Claude',
    tag: '现在连接的是：Claude 桌面版',
    guide: '写入这把密钥能用的全部 Claude 模型（Fable、Opus、Sonnet、Haiku），连接时会先关闭 Claude。',
    restoreTip: '取消中转站连接：Claude 改回用 Anthropic 官方账号登录，并关闭正在运行的 Claude。',
  },
  models: {
    label: '国产模型',
    button: '导入国产模型',
    tag: '把国产模型加入 Codex 的模型列表',
    guide: '不需要密钥。GPT、当前接口和 Key 都保留，导入后在 Codex 新建任务时选择模型。导入完成会自动关闭 Codex。',
  },
};
const CODEX_TITLES = {
  myautoapi: '已连接 MyAutoAPI',
  none: '还没连接',
  other: '当前用的是官方登录或其他服务',
};
const CLAUDE_TITLES = {
  notinstalled: '没检测到 Claude 桌面版',
  myautoapi: '已连接 MyAutoAPI',
  none: '还没连接',
  other: '当前用的是其他第三方服务',
};
const CONFIRM_MS = 6000;

let busy = false;
let tool = 'codex';
let state = { status: null, image: null, models: null, claude: null };
let confirming = null; // 'import' or 'remove' while waiting for a second click
let confirmTimer = 0;
const queuedLinks = [];

function bridge() {
  return window.go && window.go.main && window.go.main.App;
}

function waitForBridge() {
  return new Promise((resolve) => {
    const started = Date.now();
    (function check() {
      if (bridge() && window.runtime) return resolve(bridge());
      if (Date.now() - started > 5000) return resolve(bridge());
      setTimeout(check, 30);
    })();
  });
}

// The card at the top shows whichever tool is open.
function cardFor() {
  if (tool === 'image') {
    const image = state.image || { state: 'none' };
    const titles = {
      none: '未安装',
      ready: `V${image.version} 已安装`,
      nokey: `V${image.version} 已安装，缺少 Key`,
      outdated: `V${image.version} 已安装，可更新到 V${image.bundled}`,
      unmanaged: '有生图文件，但无法确认版本',
    };
    const ok = image.state === 'ready';
    return {
      state: ok ? 'ok' : image.state === 'none' ? 'none' : 'other',
      title: titles[image.state] || titles.none,
      name: image.shared ? '与 Codex 共用' : image.name || '',
      key: image.key || '',
    };
  }
  if (tool === 'models') {
    const models = state.models || { state: 'none' };
    if (models.state === 'active') return { state: 'ok', title: `已导入 ${models.count} 个国产模型`, name: '', key: '' };
    if (models.state === 'inactive') {
      return { state: 'other', title: '需要重新导入', name: '', key: '重新连接 Codex 后，国产模型要再导入一次' };
    }
    return { state: 'none', title: '还没导入', name: '', key: '' };
  }
  if (tool === 'claude') {
    const claude = state.claude || { state: 'none' };
    const connected = claude.state === 'myautoapi';
    if (claude.state === 'notinstalled') {
      return { state: 'none', title: CLAUDE_TITLES.notinstalled, name: '', key: '先装好软件，再回来点一键连接', download: true };
    }
    return {
      state: connected ? 'ok' : claude.state === 'none' ? 'none' : 'other',
      title: CLAUDE_TITLES[claude.state] || CLAUDE_TITLES.other,
      name: connected ? claude.name || '' : '',
      key: connected ? (claude.key || '') + (claude.models ? ` · ${claude.models} 个模型` : '') : '',
    };
  }
  const status = state.status || { state: 'none' };
  const connected = status.state === 'myautoapi';
  if (status.state === 'none' && state.status && !status.appInstalled) {
    return { state: 'none', title: '没检测到 Codex 软件', name: '', key: '先装好 ChatGPT（含 Codex），再回来点一键连接', download: true };
  }
  return {
    state: connected ? 'ok' : status.state === 'none' ? 'none' : 'other',
    title: CODEX_TITLES[status.state] || CODEX_TITLES.other,
    name: connected ? status.name || '' : '',
    key: connected ? status.key || '' : '',
  };
}

function renderCard() {
  const card = cardFor();
  $('status').dataset.state = card.state;
  $('status-label').textContent = TOOLS[tool].label;
  $('status-title').textContent = card.title;
  $('status-key').textContent = card.key;
  const name = $('status-name');
  name.textContent = card.name;
  name.title = card.name;
  name.hidden = card.name === '';
  $('get-app').hidden = !card.download;
  $('status-key').classList.toggle('hint', Boolean(card.download));
}

// The entries say what is already done, so nobody installs twice by mistake.
function renderEntries() {
  const image = state.image || { state: 'none' };
  const imageLabels = {
    ready: '生图工具（已安装）',
    outdated: '生图工具（可更新）',
    nokey: '生图工具（缺少 Key）',
  };
  $('go-image-label').textContent = imageLabels[image.state] || '安装生图工具';
  $('go-image').dataset.state = image.state === 'ready' ? 'done' : image.state === 'none' ? '' : 'attention';

  const models = state.models || { state: 'none' };
  let modelsLabel = '导入国产模型';
  if (models.state === 'active') modelsLabel = '国产模型（已导入）';
  else if (models.state === 'inactive') modelsLabel = '国产模型（需重新导入）';
  $('go-models-label').textContent = modelsLabel;
  $('go-models').dataset.state = models.state === 'active' ? 'done' : models.state === 'inactive' ? 'attention' : '';

  // The image tool and the domestic models belong to Codex, so the Claude
  // page shows none of the entries.
  $('tools').hidden = tool === 'claude';
  $('go-codex').hidden = tool === 'codex' || tool === 'claude';
  for (const name of ['image', 'models']) {
    $(`go-${name}`).hidden = name === tool;
  }
  for (const name of ['codex', 'claude']) {
    const pressed = name === 'claude' ? tool === 'claude' : tool !== 'claude';
    $(`tab-${name}`).setAttribute('aria-pressed', String(pressed));
  }
}

// The image tool keeps its key, so an installed one updates or reinstalls
// with it unless a new key is typed in. A link or a pasted key still
// replaces it.
function savedImageKey() {
  const image = state.image;
  return Boolean(image && image.key && (image.state === 'ready' || image.state === 'outdated'));
}

function usesSavedImageKey() {
  return tool === 'image' && savedImageKey() && keyInput.value.trim() === '';
}

function imageGuide() {
  if (!savedImageKey()) return TOOLS.image.guide;
  const image = state.image;
  const first = image.state === 'outdated'
    ? `已有密钥，直接点「一键更新」升级到 V${image.bundled}，不用再粘贴。`
    : '已装好，密钥也在。';
  return first + '要换密钥才需要粘贴新的。';
}

function guideFor(name) {
  return name === 'image' ? imageGuide() : TOOLS[name].guide;
}

function idleLabel() {
  if (usesSavedImageKey()) {
    return state.image.state === 'outdated' ? `一键更新到 V${state.image.bundled}` : '用已有密钥重新安装';
  }
  if (tool === 'models') {
    if (confirming === 'import') return '确认导入（会关闭 Codex）';
    return state.models && state.models.state === 'active' ? '重新导入国产模型' : TOOLS.models.button;
  }
  return TOOLS[tool].button;
}

function render() {
  renderCard();
  renderEntries();
  if (!busy) $('connect-label').textContent = idleLabel();
  connectButton.classList.toggle('confirm', confirming === 'import');
  removeButton.hidden = tool !== 'models' || !state.models || state.models.state === 'none';
  removeButton.textContent = confirming === 'remove' ? '确认移除（会关闭 Codex）' : '移除国产模型';
  $('restore-anchor').hidden = tool !== 'codex' && tool !== 'claude';
  if (TOOLS[tool].restoreTip) $('restore-tip').textContent = TOOLS[tool].restoreTip;
}

function absorb(result) {
  if (result.status) state.status = result.status;
  if (result.image) state.image = result.image;
  if (result.models) state.models = result.models;
  if (result.claude) state.claude = result.claude;
}

function showNotice(tone, text, action, tag) {
  const notice = $('notice');
  notice.dataset.tone = tone;
  const paragraph = $('notice-text');
  paragraph.textContent = text;
  if (tag) {
    const strong = document.createElement('strong');
    strong.className = 'mode-tag';
    strong.textContent = tag;
    paragraph.prepend(strong);
  }
  $('notice-action').hidden = action !== 'open-site';
  notice.hidden = false;
}

function showResult(result) {
  absorb(result);
  render();
  showNotice(result.tone, result.message, result.action);
}

function setBusy(on, label) {
  busy = on;
  for (const button of [connectButton, restoreButton, removeButton, $('go-codex'), $('go-image'), $('go-models'),
    $('tab-codex'), $('tab-claude'), $('get-official'), $('get-group')]) {
    button.disabled = on;
  }
  connectButton.classList.toggle('busy', on);
  $('connect-label').textContent = on ? label : idleLabel();
}

function cancelConfirm() {
  clearTimeout(confirmTimer);
  confirming = null;
}

function askConfirm(what) {
  cancelConfirm();
  confirming = what;
  confirmTimer = setTimeout(() => {
    confirming = null;
    render();
  }, CONFIRM_MS);
  render();
}

function setTool(next) {
  if (next === tool) return;
  cancelConfirm();
  tool = next;
  shell.dataset.tool = tool;
  keyInput.value = '';
  keyInput.placeholder = tool === 'image' && savedImageKey() ? '已有密钥；要换密钥才粘贴新的' : TOOLS[tool].placeholder || '';
  shell.classList.remove('activating');
  void shell.offsetWidth; // restart the sweep
  shell.classList.add('activating');
  setTimeout(() => shell.classList.remove('activating'), 950);
  render();
  if (tool === 'codex') {
    $('notice').hidden = true;
  } else {
    const openSite = tool === 'image' && !savedImageKey() ? 'open-site' : '';
    showNotice(tool, guideFor(tool), openSite, TOOLS[tool].tag);
  }
  if ((tool === 'image' && !savedImageKey()) || tool === 'claude') keyInput.focus();
  if (tool === 'claude' || tool === 'codex') fillFromClipboard();
}

async function run(label, notice, action) {
  setBusy(true, label);
  showNotice('info', notice);
  try {
    const result = await action();
    // A finished image install or model import goes back to Codex; the
    // result stays on screen. Claude stays on its own tab.
    if ((tool === 'image' || tool === 'models') && result.tone !== 'error') setTool('codex');
    showResult(result);
  } catch (error) {
    showNotice('error', '操作没有完成，请再试一次。');
  } finally {
    setBusy(false);
    nextLink();
  }
}

function submit() {
  if (tool === 'models') {
    if (confirming !== 'import') {
      askConfirm('import');
      return;
    }
    cancelConfirm();
    run('正在导入…', '正在导入国产模型…', () => bridge().ImportModels());
    return;
  }
  const key = keyInput.value;
  if (usesSavedImageKey()) {
    run('正在更新…', '正在用已有密钥更新生图工具…', () => bridge().UpdateImage());
    return;
  }
  if (tool === 'image') {
    run('正在安装…', '正在检查密钥能否画图…', () => bridge().InstallImage(key));
    return;
  }
  if (tool === 'claude') {
    run('正在连接…', '正在检查密钥和可用的 Claude 模型…', () => bridge().ConnectClaude(key));
    return;
  }
  run('正在连接…', '正在检查密钥…', () => bridge().Connect(key));
}

// Links import straight away. An app=claude link connects Claude Desktop.
// An app=codex link goes to the image tool while it is open; anywhere else it
// connects Codex, as it always has.
async function importLink(raw) {
  setBusy(true, '正在连接…');
  let parsed = null;
  try {
    parsed = await bridge().ParseLink(raw);
  } catch (error) {
    parsed = null;
  }
  if (!parsed || !parsed.ok) {
    setBusy(false);
    if (parsed) showResult(parsed.result);
    else showNotice('error', '操作没有完成，请再试一次。');
    nextLink();
    return;
  }
  const name = parsed.name || '';
  if (parsed.app === 'claude') {
    if (tool !== 'claude') setTool('claude');
    keyInput.value = parsed.key;
    const notice = name ? `正在把「${name}」接入 Claude…` : '正在把密钥接入 Claude…';
    await run('正在连接…', notice, () => bridge().ImportClaude(parsed.key, name));
    return;
  }
  if (tool === 'image') {
    keyInput.value = parsed.key;
    const notice = name ? `正在把「${name}」导入生图工具…` : '正在把密钥导入生图工具…';
    await run('正在安装…', notice, () => bridge().ImportImage(parsed.key, name));
    return;
  }
  if (tool !== 'codex') setTool('codex');
  keyInput.value = parsed.key;
  const notice = name ? `正在导入「${name}」…` : '正在导入密钥…';
  await run('正在连接…', notice, () => bridge().Import(parsed.key, name));
}

function enqueueLink(raw) {
  if (!raw) return;
  queuedLinks.push(raw);
  nextLink();
}

function nextLink() {
  if (busy || queuedLinks.length === 0) return;
  importLink(queuedLinks.shift());
}

async function fillFromClipboard() {
  // An installed image tool already has its key; the clipboard might hold
  // the Codex key, which must not replace it unasked.
  if (busy || tool === 'models' || (tool === 'image' && savedImageKey()) || keyInput.value.trim() !== '' || !bridge()) return;
  const asked = tool;
  try {
    const key = await bridge().ClipboardKey();
    // The window may have moved on while the clipboard was read.
    if (!key || busy || tool !== asked || keyInput.value.trim() !== '') return;
    keyInput.value = key;
    $('connect-label').textContent = idleLabel();
  } catch (error) {
    // The clipboard may be busy; the user can still paste by hand.
  }
}

// The button says whether the saved image key or the typed one is used.
keyInput.addEventListener('input', () => {
  if (!busy) $('connect-label').textContent = idleLabel();
});

$('connect-form').addEventListener('submit', (event) => {
  event.preventDefault();
  if (!busy) submit();
});

for (const name of ['codex', 'image', 'models']) {
  $(`go-${name}`).addEventListener('click', () => {
    if (!busy) setTool(name);
  });
}

for (const name of ['codex', 'claude']) {
  $(`tab-${name}`).addEventListener('click', () => {
    tabsUsed = true;
    if (!busy) setTool(name);
  });
}

// New users may not notice the two tabs, so each one pulses once: Codex 5
// seconds after opening, Claude after 20. Nothing pulses once a tab was
// clicked, while working, or for the page already open (Claude).
const TAB_HINTS = [['codex', 5000], ['claude', 20000]];
let tabsUsed = false;

function hintTab(name) {
  if (tabsUsed || busy || (name === 'claude' && tool === 'claude') || (name === 'codex' && tool !== 'codex')) return;
  const tab = $(`tab-${name}`);
  tab.classList.remove('hint');
  void tab.offsetWidth;
  tab.classList.add('hint');
  tab.addEventListener('animationend', () => tab.classList.remove('hint'), { once: true });
}

$('get-official').addEventListener('click', () => bridge().OpenDownload(tool === 'claude' ? 'claude' : 'codex'));

// The QQ group's files carry both installers, for anyone who cannot reach
// the official sites.
$('get-group').addEventListener('click', () => {
  const qq = $('qq');
  qq.classList.remove('flash');
  void qq.offsetWidth;
  qq.classList.add('flash');
  const app = tool === 'claude' ? 'Claude' : 'ChatGPT';
  showNotice('info', `打不开官网时，进 QQ 群在「群文件」里下载 ${app} 安装包：点下方「加入群聊」，或复制群号 ${$('version').dataset.qq || '685861462'} 搜索加群。`);
});

removeButton.addEventListener('click', () => {
  if (busy) return;
  if (confirming !== 'remove') {
    askConfirm('remove');
    return;
  }
  cancelConfirm();
  run('正在移除…', '正在移除国产模型…', () => bridge().RemoveModels());
});

restoreButton.addEventListener('click', () => {
  if (busy) return;
  if (tool === 'claude') run('正在恢复…', '正在恢复 Claude 官方登录…', () => bridge().RestoreClaude());
  else run('正在恢复…', '正在恢复官方登录…', () => bridge().Restore());
});

$('copy-qq').addEventListener('click', async (event) => {
  const button = event.currentTarget;
  if (await bridge().CopyGroupNumber()) {
    button.textContent = '已复制';
    button.classList.add('done');
    setTimeout(() => {
      button.textContent = '复制群号';
      button.classList.remove('done');
    }, 1600);
  }
});

$('notice-action').addEventListener('click', () => bridge().OpenSite());
$('join-qq').addEventListener('click', () => bridge().JoinGroup());
$('open-site').addEventListener('click', () => bridge().OpenSite());

window.addEventListener('focus', fillFromClipboard);
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') fillFromClipboard();
});

(async function start() {
  const app = await waitForBridge();
  if (!app) return;
  const info = await app.Info();
  $('version').textContent = info.version;
  $('version').dataset.qq = info.qq;
  state.status = await app.Status();
  state.image = await app.ImageStatus();
  state.models = await app.ModelsStatus();
  state.claude = await app.ClaudeStatus();
  render();
  for (const [name, delay] of TAB_HINTS) setTimeout(() => hintTab(name), delay);
  window.runtime.EventsOn('import-link', enqueueLink);
  const link = await app.TakeLaunchLink();
  if (link) {
    enqueueLink(link);
  } else {
    fillFromClipboard();
  }
})();
