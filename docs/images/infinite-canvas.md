# 无限画布

无限画布是浏览器里的图片创作画布：提示词、参考图和生成结果放在同一张画布上，适合角色设定、产品图、海报草图这类需要多轮修改的创作。全程在浏览器完成，不改电脑上的文件。

访问地址：<https://canvas.myautoapi.com>。只能用 **专门画图分组** 的 API Key 接入，其它分组的 Key 不能用；创建方法看 [安装生图工具第 1 步](/image-video-group-image#_1-创建专门画图分组的-api-key)。

## 1. 打开设置

点击页面右上角的滑杆图标，打开设置弹窗。

<details>
<summary>查看设置入口截图</summary>

<a class="doc-image-link" href="/images/画图/画图1.jpg" target="_blank" rel="noopener">
  <img src="/images/画图/画图1.jpg" alt="无限画布右上角设置入口">
</a>

</details>

## 2. 填写渠道

停在 **渠道** 页，填写：

| 项目 | 填写 |
| --- | --- |
| 调用格式 | `OpenAI` |
| 渠道名称 | `MyAutoAPI` |
| API Key | 专门画图分组的 MyAutoAPI API Key |
| API 地址 | `https://api.myautoapi.com`，后面不加 `/v1` |

填好后点击 **拉取全部**，画布会读取这个 Key 所属分组里的可用模型。

<details>
<summary>查看渠道页截图</summary>

<a class="doc-image-link" href="/images/画图/画图2.jpg" target="_blank" rel="noopener">
  <img src="/images/画图/画图2.jpg" alt="无限画布渠道页填写 Base URL 和 API Key">
</a>

</details>

API Key 保存在当前浏览器里，不要在公共电脑上填写。

## 3. 选择默认模型

切到 **模型** 页，从下拉框选择：

| 项目 | 选择 |
| --- | --- |
| 默认生图模型 | `gpt-image-2` |
| 默认文本模型 | 下拉框里列出的文本模型 |

点击 **完成** 保存。

<details>
<summary>查看模型页截图</summary>

<a class="doc-image-link" href="/images/画图/画图3.jpg" target="_blank" rel="noopener">
  <img src="/images/画图/画图3.jpg" alt="无限画布模型页选择默认生图模型和默认文本模型">
</a>

</details>

## 4. 生成第一张图

回到 **生图工作台**，输入 `一只透明玻璃杯，白底，柔和自然光`，生成。图片出现在画布上即配置完成。

## 使用须知

- 一张图需要 2 到 15 分钟。生成中断网、刷新或关闭页面，这次生成会失败。
- 画布内容和图片只保存在当前浏览器。换浏览器、清理站点数据或用无痕窗口会看不到原来的内容，重要图片生成后及时下载。
- 多开几个页面可以并行作图，每个页面等图片完成后再关闭。

## 常见问题

| 遇到的问题 | 处理方法 |
| --- | --- |
| 保存设置后没有反应 | 刷新页面，重新打开设置，确认渠道和模型还在 |
| 拉取不到模型或图片不能生成 | 确认 Key 属于专门画图分组，默认生图模型选 `gpt-image-2`，重新点击 **拉取全部** |
| 浏览器报连接失败 | 关闭会拦截请求的浏览器插件，换网络或换一个浏览器再试 |
