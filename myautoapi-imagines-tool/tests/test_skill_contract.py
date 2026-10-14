"""Contract checks for the installed Skill template and user-facing metadata."""

from __future__ import annotations

import re
import unittest
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
SKILL_PATH = PROJECT_ROOT / "skill-template" / "myautoapi-image-generation" / "SKILL.md"
OPENAI_YAML_PATH = SKILL_PATH.parent / "agents" / "openai.yaml"
AGENTS_RULE_PATH = PROJECT_ROOT / "installer" / "AGENTS.rule.md"


class SkillContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.skill = SKILL_PATH.read_text(encoding="utf-8")
        cls.openai_yaml = OPENAI_YAML_PATH.read_text(encoding="utf-8")
        cls.agents_rule = AGENTS_RULE_PATH.read_text(encoding="utf-8")

    def test_description_triggers_without_brand_or_model_name(self) -> None:
        self.assertRegex(
            self.skill,
            r"description: .*generate.*draw.*make.*regenerate.*compare.*edit.*modify.*reference.*multiple",
        )
        self.assertIn("用户不需要说出 MyAutoAPI", self.skill)

    def test_skill_uses_fixed_tool_contract(self) -> None:
        self.assertIn("gpt-image-2", self.skill)
        self.assertIn("gpt-image-1k-th", self.skill)
        self.assertIn("gpt-image-2-adobe", self.skill)
        self.assertIn("grok-imagine-image", self.skill)
        self.assertIn("nano-banana-2", self.skill)
        self.assertIn("nano-banana-pro", self.skill)
        self.assertIn("https://api.myautoapi.com/v1/images/generations", self.skill)
        self.assertIn("https://api.myautoapi.com/v1/responses", self.skill)
        self.assertIn('--model "$model"', self.skill)
        self.assertIn('--size "$size"', self.skill)
        self.assertIn('--quality "$quality"', self.skill)
        self.assertIn("--output-name \"准确的图片名称\"", self.skill)
        self.assertIn("项目根目录的 `images`", self.skill)
        self.assertIn("桌面的 `images`", self.skill)
        self.assertIn("[Environment+SpecialFolder]::Desktop", self.skill)
        self.assertIn("桌面 `images` 或输入图片读取权限", self.skill)
        self.assertIn("用户不批准时停止", self.skill)
        self.assertIn('--output-dir "$output_dir"', self.skill)
        self.assertIn('--output-dir "$OutputDir"', self.skill)
        self.assertNotIn("$CODEX_HOME/images", self.skill)
        self.assertNotIn("myautoapi-image-results", self.skill)
        self.assertNotIn("record_dir", self.skill)

    def test_skill_lists_models_and_keeps_comparison_user_driven(self) -> None:
        for phrase in (
            "用户问有哪些模型",
            "`gpt-image-2`",
            "`gpt-image-1k-th`",
            "`gpt-image-2-adobe`",
            "`grok-imagine-image`",
            "`nano-banana-2`",
            "`nano-banana-pro`",
            "同一个提示词分别生成",
            "只有用户明确同意后",
            "不自动评价或排名",
            "`1024x1024`、`high`（也支持 `low`）",
            "`1024x1536`",
            "`1536x2304`",
            "`2304x3456`",
            "`--resolution 1k`",
            "`--resolution 2k`",
            "`--aspect-ratio 16:9`",
            "`--resolution 4k`",
            "PNG、JPEG、WebP 各 1 张参考图",
            "2 张和 4 张不同参考图",
            "14 张尚未形成远端支持结论",
            "`1:8`",
        ):
            self.assertIn(phrase, self.skill)
        self.assertNotIn("最多传入 14 张 PNG、JPEG 或 WebP", self.skill)
        for unavailable in (
            "gpt-image-1k-adobe",
            "gpt-image-2k-adobe",
            "gpt-image-4k-adobe",
            "ad-gpt-image-2-mx",
        ):
            self.assertNotIn(unavailable, self.skill)

    def test_skill_routes_gpt_image_25_models_and_defaults_to_2k(self) -> None:
        routing_rule = (
            "模型选择顺序固定为：用户明确指定模型时使用该模型；未指定模型且明确要求高清、高质量、"
            "精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`；"
            "其余未指定模型的任务默认使用 `gpt-image-2.5-flare`。"
        )
        for content in (self.skill, self.agents_rule):
            for phrase in (
                "`gpt-image-2.5-flare`",
                "`gpt-image-2.5-sunburst`",
                "默认使用 `gpt-image-2.5-flare`",
                "`2048x2048`",
                "明确要求高清、高质量、精细细节、精修、严格保留原图或重要成品时使用 `gpt-image-2.5-sunburst`",
                "用户明确指定模型时使用该模型",
            ):
                self.assertIn(phrase, content)
            self.assertIn(routing_rule, content)

        for quality in ("`xhigh`", "`max`"):
            self.assertIn(quality, self.skill)
        self.assertIn("只适用于 `gpt-image-2.5-flare` 和 `gpt-image-2.5-sunburst`", self.skill)
        self.assertNotIn('model="gpt-image-2.5-flare"', self.skill)
        self.assertNotIn("--model gpt-image-2.5-flare", self.skill)
        self.assertNotIn('$Model = "gpt-image-2.5-flare"', self.skill)

    def test_skill_contains_complete_text_to_image_parameter_guidance(self) -> None:
        for value in (
            "1024x1024",
            "1536x1024",
            "1024x1536",
            "2048x2048",
            "2048x1152",
            "1152x2048",
            "3840x2160",
            "2160x3840",
            "low",
            "medium",
            "high",
            "auto",
        ):
            self.assertIn(value, self.skill)
        for phrase in (
            "下表是建议值，不是替用户作创作决定",
            "宽高都是 16 的倍数",
            "长边不超过短边 3 倍",
            "按接口返回的真实图片格式保存",
        ):
            self.assertIn(phrase, self.skill)

    def test_skill_protects_key_and_displays_saved_paths_directly(self) -> None:
        for phrase in (
            "绝不打开、读取、列出、复制、打印、传递或扫描 `auth.json`",
            "单模型普通任务只运行一次工具",
            "也不调用 `view_image` 或其它工具重新读取图片",
            "立即使用绝对路径 Markdown 图片语法直接展示",
            "不换模型、不伪造图片",
        ):
            self.assertIn(phrase, self.skill)
        self.assertNotRegex(self.skill, r"--api-key|api_key\s*=|sk-[A-Za-z0-9]{20,}")
        self.assertIn("tools/myautoapi-imagines-tool/api-key", self.skill)

    def test_skill_limits_model_discovery_to_explicit_low_frequency_cases(self) -> None:
        for phrase in (
            "低频模型探测",
            "当前/最新有哪些模型",
            "同一轮任务只探测一次",
            "普通生图、Skill 加载、启动 Codex",
            "GET https://api.myautoapi.com/v1/models",
            "不得自动加入白名单",
            "只输出 HTTP 状态和 `id`/`display_name`",
            "Authorization: Bearer",
            "未更新模型列表",
        ):
            self.assertIn(phrase, self.skill)
        self.assertNotIn("每次生图前探测", self.skill)

    def test_skill_parallelizes_models_without_detached_background_jobs(self) -> None:
        for phrase in (
            "同一批并发工具调用中同时提交",
            "每个模型只运行一次",
            "独立前台调用",
            "不能因为“每个模型需要独立前台调用”而改成顺序执行",
            "禁止用 shell 后台符号",
            "PowerShell `Start-Job`",
            "继续等待这一批调用全部完成",
            "模型、工具调用和 stdout 结果的对应关系",
            "不使用其它命令检查进程",
            "正常取得 stdout 时不扫描 `images` 目录",
            "生图调用不要使用 `Get-ChildItem`、`Get-Content`、`Get-Process`、`Test-Path` 或 `mkdir`",
            "宿主为加载本 `SKILL.md` 进行的一次读取属于 Skill 加载",
        ):
            self.assertIn(phrase, self.skill)
        for phrase in (
            "使用宿主的并发工具调用能力在同一批中同时提交",
            "不得因需要独立调用而改成顺序执行",
            "多模型时等待整批调用全部退出",
        ):
            self.assertIn(phrase, self.agents_rule)

    def test_bash_code_mode_forwards_nested_result_and_recovers_only_when_needed(self) -> None:
        for phrase in (
            "Linux 或 macOS Code Mode 结果转发",
            "while (result.session_id)",
            "text(JSON.stringify",
            "exit_code: result.exit_code ?? null",
            "output += result.output ??",
            "session_id: null",
            "tools.write_stdin",
            "累积每次返回的 `output`",
            "正常取得 stdout 时不扫描 `images` 目录",
            "外层却没有取得可解析的成功或失败 JSON",
            "本次已知的 `output_dir`",
            "修改时间不早于本次调用开始时间",
            "不能只按“最近 N 张”选择文件",
            "多于 `n` 时归属不明确",
            "具体模型归属无法确认",
            "不得重新生图、更新或重装工具",
            "不得据此归因于中转站、macOS 权限或工具保存失败",
        ):
            self.assertIn(phrase, self.skill)
        for phrase in (
            "必须用 `text(...)` 显式转发",
            "Windows PowerShell 调用流程不变",
            "正常取得 stdout 时不扫描输出目录",
            "不能只取最近 N 张",
            "不重新生图、不更新或重装工具",
        ):
            self.assertIn(phrase, self.agents_rule)
        self.assertNotIn("receipt", self.skill.lower())
        self.assertNotIn("收据", self.skill)

    def test_bash_code_mode_collects_the_same_session_without_waiting_for_user(self) -> None:
        for phrase in (
            "Linux 或 macOS Code Mode 结果转发",
            '// @exec: {"yield_time_ms": 120000, "max_output_tokens": 4000}',
            "yield_time_ms: 5000",
            "const startedAt = Date.now()",
            "let nextStatusAt = startedAt + 30000",
            "result.session_id && Date.now() >= nextStatusAt",
            'notify("图片仍在生成，继续等待同一个调用。")',
            "nextStatusAt = Date.now() + 30000",
            "每 5 秒",
            "进程退出时会立即返回",
            "每累计 30 秒",
            "不等待用户再次发消息或催促",
            "外层仍返回 `Script running with cell ID ...`",
            "立即对同一个 `cell_id` 调用 `functions.wait`",
            "`yield_time_ms: 120000`",
        ):
            self.assertIn(phrase, self.skill)
        for phrase in (
            "Linux 或 macOS Code Mode",
            "每 5 秒收取同一个 `session_id`",
            "每累计 30 秒",
            "不等待用户再次发消息或催促",
            "外层返回 `Script running with cell ID ...`",
            "立即对同一个 `cell_id` 调用 `functions.wait`",
        ):
            self.assertIn(phrase, self.agents_rule)

        code_mode_section = self.skill.split("### Linux 或 macOS Code Mode 结果转发", 1)[1]
        code_mode_example = code_mode_section.split("Windows PowerShell：", 1)[0]
        javascript_example = code_mode_example.split("```javascript", 1)[1].split("```", 1)[0]
        self.assertTrue(
            javascript_example.lstrip().startswith(
                '// @exec: {"yield_time_ms": 120000, "max_output_tokens": 4000}'
            )
        )
        self.assertEqual(javascript_example.count("yield_time_ms: 5000"), 2)
        self.assertEqual(javascript_example.count("notify("), 1)
        self.assertNotIn("nextStatusAt += 30000", javascript_example)
        self.assertNotIn("yield_time_ms: 30000", javascript_example)

    def test_windows_command_stays_out_of_bash_code_mode_fallback(self) -> None:
        code_mode_tail = self.skill.split("### Linux 或 macOS Code Mode 结果转发", 1)[1]
        windows_section = code_mode_tail.split("Windows PowerShell：", 1)[1].split(
            "## 结果处理", 1
        )[0]
        self.assertIn("& $ToolPath generate", windows_section)
        self.assertIn("--output-dir \"$OutputDir\"", windows_section)
        self.assertNotIn("exec_command", windows_section)
        self.assertNotIn("恢复核查", windows_section)

    def test_nano_banana_example_does_not_pass_quality(self) -> None:
        nano_section = self.skill.split("macOS Nano Banana 文生图、多图、参考图或修图：", 1)[1]
        nano_section = nano_section.split("### Linux 或 macOS Code Mode 结果转发", 1)[0]
        self.assertNotIn("--quality", nano_section)

    def test_skill_keeps_only_flat_final_images(self) -> None:
        for phrase in (
            "只在 `images` 下保存最终图片",
            "不创建图片专属子目录",
            "单图重名时自动追加数字",
            "多图使用 `名称-1`、`名称-2`、`名称-3`",
            "不要使用时间戳、模型名或泛化的 `result`",
        ):
            self.assertIn(phrase, self.skill)
        for obsolete in (
            "request.json",
            "response.json",
            "response.raw",
            "summary.json",
            "record_dir",
        ):
            self.assertNotIn(obsolete, self.skill)

    def test_skill_requests_narrow_reusable_network_permission(self) -> None:
        for content in (self.skill, self.agents_rule):
            for phrase in (
                "不要先在无网络权限的沙箱中试跑",
                "原生工具的 `generate` 或 `edit` 子命令申请可复用的最小许可",
                "不能扩大到整个 PowerShell、Bash 或其它命令",
                "不要求用户开启“完全访问”",
                "不修改 `config.toml`",
                "用户拒绝",
            ):
                self.assertIn(phrase, content)

    def test_skill_keeps_user_messages_chinese_and_waits_for_exit(self) -> None:
        for content in (self.skill, self.agents_rule):
            for phrase in (
                "全部使用中文",
                "不输出英文进度说明",
                "任一调用未退出时不得结束当前轮次",
                "不得声称图片稍后会自动显示",
            ):
                self.assertIn(phrase, content)
        self.assertIn("不得把“正在运行”当成最终结果", self.skill)

    def test_powershell_prompt_is_passed_as_a_here_string_variable(self) -> None:
        for content in (self.skill, self.agents_rule):
            self.assertIn("单引号 here-string", content)
            self.assertIn("作为 `--prompt` 的独立参数", content)
        self.assertIn("$Prompt = @'", self.skill)
        self.assertIn("--prompt $Prompt", self.skill)
        self.assertIn("--model $Model", self.skill)
        powershell_example = self.skill.split("```powershell", 1)[1].split("```", 1)[0]
        self.assertNotIn('--prompt "用户给出的完整图片描述"', powershell_example)

    def test_global_rule_has_explicit_manual_update_entry(self) -> None:
        for phrase in (
            "只有用户明确说“更新生图工具”“升级生图工具”或要求重装时才执行更新",
            "用户要求修复时先捕捉本机真实原因，不把更新当成修复",
            "irm https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.ps1 | iex",
            "curl -fsSL https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh | bash",
            "不创建后台服务、计划任务、开机启动",
            "每次执行都输入并覆盖 `api-key`",
        ):
            self.assertIn(phrase, self.agents_rule)
        self.assertIn(
            "Linux x64、Apple Silicon 或 Intel Mac 使用 `curl -fsSL https://docs.myautoapi.com/install/myautoapi-imagines-tool/install.sh | bash`",
            self.agents_rule,
        )

    def test_installed_context_does_not_expose_uninstall_instructions(self) -> None:
        for content in (self.skill, self.agents_rule):
            self.assertNotIn("卸载", content)
            self.assertNotIn("uninstall", content.lower())

    def test_skill_has_both_platform_entries_and_metadata(self) -> None:
        self.assertIn("myautoapi-imagines-tool\" generate", self.skill)
        self.assertIn("myautoapi-imagines-tool\" edit", self.skill)
        self.assertIn("myautoapi-imagines-tool.exe", self.skill)
        self.assertNotIn("python3", self.skill)
        self.assertNotIn("py $ToolScript", self.skill)
        self.assertIn("display_name: \"中转站生图与修图\"", self.openai_yaml)
        self.assertIn("$myautoapi-image-generation", self.openai_yaml)
        self.assertIn("Explicitly forward nested Linux and macOS Code Mode results", self.openai_yaml)
        self.assertIn("recover only attributable files", self.openai_yaml)
        self.assertIn("mapped to exact model IDs", self.openai_yaml)
        self.assertIn("allow_implicit_invocation", self.openai_yaml)

    def test_skill_handles_reference_edit_and_multi_image_display(self) -> None:
        for phrase in (
            "https://api.myautoapi.com/v1/images/edits",
            "不要把参考图自动当成修改目标",
            "第一张 `--image` 是修改目标",
            "通过重复的 `--image` 传入",
            "修改图片始终另存为新图片",
            "使用一次 `generate --n N`",
            "通用数量为 1 到 10，Grok Imagine 为 1 到 9",
            "所有模型的多图都由工具内部拆成独立单张请求",
            "并发上限为 3",
            "多张成功使用 `output_files`",
            "`output_count` 是实际成功数量",
            "也不调用 `view_image` 或其它工具重新读取图片",
            "`模型：<准确模型 ID>`",
            "`第 i/output_count 张`",
            "一个模型完全失败时仍展示其它成功模型的全部图片",
            "最终回复必须再次使用绝对路径 Markdown 图片语法嵌入",
            "`![生成结果](/absolute/path/image.png)`",
            "最终回复中的嵌入数量与 `output_count` 一致",
            "已生成 x/N 张。",
        ):
            self.assertIn(phrase, self.skill)
        self.assertIn("`output_count` 是实际成功数量", self.agents_rule)
        self.assertIn("准确模型 ID", self.agents_rule)
        self.assertIn("不调用 `view_image` 或其它工具重新读取图片", self.agents_rule)
        self.assertIn("最终回复必须再次嵌入全部成功图片", self.agents_rule)
        self.assertIn("嵌入数量与 `output_count` 一致", self.agents_rule)


if __name__ == "__main__":
    unittest.main()
