"""V0.1 contract tests for the local MyAutoAPI image tool.

These tests deliberately use only fake transports and a tiny PNG fixture. They
must never need a real API key or send a network request.
"""

from __future__ import annotations

import base64
import importlib.util
import json
import shutil
import sys
import tempfile
import unittest
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
TOOL_PATH = PROJECT_ROOT / "tool" / "生成图片.py"
MODULE_SPEC = importlib.util.spec_from_file_location("myautoapi_image_tool", TOOL_PATH)
assert MODULE_SPEC is not None and MODULE_SPEC.loader is not None
tool = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = tool
MODULE_SPEC.loader.exec_module(tool)


# 1 x 1 opaque PNG. The bytes are intentionally small so every test is local.
PNG_BYTES = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/" \
    "T6R2JQAAAABJRU5ErkJggg=="
)
PNG_B64 = base64.b64encode(PNG_BYTES).decode("ascii")
FAKE_KEY = "not-a-real-key-must-never-be-written"


class GenerateImageContractTests(unittest.TestCase):
    def test_build_payload_has_explicit_v01_defaults(self) -> None:
        self.assertEqual(
            tool.build_payload("一颗红苹果", "1024x1024", "high", "png"),
            {
                "model": "gpt-image-2",
                "prompt": "一颗红苹果",
                "n": 1,
                "size": "1024x1024",
                "quality": "high",
                "output_format": "png",
            },
        )

    def test_invalid_model_and_prompt_are_rejected(self) -> None:
        with self.assertRaises(tool.InputError):
            tool.build_payload("", "1024x1024", "high", "png")
        with self.assertRaises(tool.InputError):
            tool.build_payload("  ", "1024x1024", "high", "png")
        with self.assertRaises(tool.InputError):
            tool.validate_model("gpt-image-1")

    def test_codex_home_and_auth_path_are_fixed(self) -> None:
        environment = {"CODEX_HOME": "/tmp/codex-home"}
        self.assertEqual(tool.codex_home(environment), Path("/tmp/codex-home"))
        self.assertEqual(
            tool.auth_path(environment),
            Path("/tmp/codex-home/auth.json"),
        )
        self.assertEqual(
            tool.default_output_dir(environment),
            Path("/tmp/codex-home/images"),
        )
        self.assertEqual(tool.codex_home({}), Path.home() / ".codex")

    def test_missing_or_invalid_auth_does_not_echo_secret(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            environment = {"CODEX_HOME": temporary_directory}
            with self.assertRaises(tool.ToolError) as missing:
                tool.load_api_key(environment)
            self.assertIn("auth.json", str(missing.exception))
            self.assertNotIn(FAKE_KEY, str(missing.exception))

            auth_path = tool.auth_path(environment)
            auth_path.write_text("{}", encoding="utf-8")
            with self.assertRaises(tool.ToolError) as missing_field:
                tool.load_api_key(environment)
            self.assertIn("OPENAI_API_KEY", str(missing_field.exception))
            self.assertNotIn(FAKE_KEY, str(missing_field.exception))

            auth_path.write_text('{"OPENAI_API_KEY":""}', encoding="utf-8")
            with self.assertRaises(tool.ToolError) as empty_field:
                tool.load_api_key(environment)
            self.assertIn("OPENAI_API_KEY", str(empty_field.exception))
            self.assertNotIn(FAKE_KEY, str(empty_field.exception))

    def test_b64_json_wins_over_url_without_download(self) -> None:
        downloader_calls: list[str] = []

        def transport(_request: bytes, _key: str) -> tool.HttpResponse:
            return tool.HttpResponse(
                200,
                {"content-type": "application/json"},
                json.dumps(
                    {"data": [{"b64_json": PNG_B64, "url": "https://example.invalid/image.png"}]}
                ).encode("utf-8"),
            )

        def downloader(url: str) -> tool.HttpResponse:
            downloader_calls.append(url)
            return tool.HttpResponse(200, {}, PNG_BYTES)

        summary, records = self.run_with_fake_key(transport, downloader)
        self.assertTrue(summary["ok"])
        self.assertEqual(summary["source"], "b64_json")
        self.assertEqual(summary["format"], "png")
        self.assertEqual(summary["dimensions"], [1, 1])
        self.assertEqual(downloader_calls, [])
        self.assertEqual(Path(summary["output_file"]).suffix, ".png")
        self.assertEqual(Path(summary["output_file"]).read_bytes(), PNG_BYTES)
        self.assert_record_files(records, expect_image=True)

    def test_default_output_dir_uses_codex_home_images(self) -> None:
        def transport(_request: bytes, _key: str) -> tool.HttpResponse:
            return tool.HttpResponse(
                200,
                {"content-type": "application/json"},
                json.dumps({"data": [{"b64_json": PNG_B64}]}).encode("utf-8"),
            )

        with tempfile.TemporaryDirectory() as temporary_directory:
            environment = self.write_fake_key(temporary_directory)
            summary = tool.run_generation(
                prompt="一颗红苹果",
                environ=environment,
                transport=transport,
            )
            records = Path(summary["record_dir"])
            self.assertEqual(records.parent, Path(temporary_directory) / "images")
            self.assert_record_files(records, expect_image=True)

    def test_url_fallback_downloads_exactly_once(self) -> None:
        downloader_calls: list[str] = []

        def transport(_request: bytes, _key: str) -> tool.HttpResponse:
            return tool.HttpResponse(
                200,
                {},
                json.dumps({"data": [{"url": "https://example.invalid/image.png"}]}).encode("utf-8"),
            )

        def downloader(url: str) -> tool.HttpResponse:
            downloader_calls.append(url)
            return tool.HttpResponse(200, {"content-type": "image/png"}, PNG_BYTES)

        summary, records = self.run_with_fake_key(transport, downloader)
        self.assertTrue(summary["ok"])
        self.assertEqual(summary["source"], "url")
        self.assertEqual(downloader_calls, ["https://example.invalid/image.png"])
        self.assertTrue((records / "image-headers.json").is_file())
        self.assert_record_files(records, expect_image=True)

    def test_invalid_b64_falls_back_to_url(self) -> None:
        downloader_calls: list[str] = []

        def transport(_request: bytes, _key: str) -> tool.HttpResponse:
            return tool.HttpResponse(
                200,
                {},
                json.dumps(
                    {"data": [{"b64_json": "not base64", "url": "https://example.invalid/image.png"}]}
                ).encode("utf-8"),
            )

        def downloader(url: str) -> tool.HttpResponse:
            downloader_calls.append(url)
            return tool.HttpResponse(200, {}, PNG_BYTES)

        summary, _records = self.run_with_fake_key(transport, downloader)
        self.assertEqual(summary["source"], "url")
        self.assertEqual(downloader_calls, ["https://example.invalid/image.png"])

    def test_http_error_saves_evidence_without_result(self) -> None:
        def transport(_request: bytes, _key: str) -> tool.HttpResponse:
            return tool.HttpResponse(429, {"content-type": "application/json"}, b'{"error":"busy"}')

        with tempfile.TemporaryDirectory() as temporary_directory:
            environment = self.write_fake_key(temporary_directory)
            with self.assertRaises(tool.GenerationError) as failure:
                tool.run_generation(
                    prompt="一颗红苹果",
                    size="1024x1024",
                    quality="high",
                    output_format="png",
                    output_dir=Path(temporary_directory) / "results",
                    environ=environment,
                    transport=transport,
                )
            records = failure.exception.record_dir
            self.assertIsNotNone(records)
            assert records is not None
            self.assert_record_files(records, expect_image=False)
            self.assertEqual(list(records.glob("result.*")), [])
            summary = json.loads((records / "summary.json").read_text(encoding="utf-8"))
            self.assertFalse(summary["ok"])
            self.assertEqual(summary["status"], 429)

    def test_cli_defaults_and_rejections(self) -> None:
        parser = tool.create_parser()
        arguments = parser.parse_args(["--prompt", "一颗红苹果"])
        self.assertEqual(arguments.model, "gpt-image-2")
        self.assertEqual(arguments.size, "1024x1024")
        self.assertEqual(arguments.quality, "high")
        self.assertEqual(arguments.output_format, "png")
        self.assertIsNone(arguments.output_dir)
        self.assertEqual(tool.main(["--model", "other", "--prompt", "x"]), 1)

    def run_with_fake_key(self, transport, downloader):
        temporary_directory = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, temporary_directory, ignore_errors=True)
        environment = self.write_fake_key(temporary_directory)
        summary = tool.run_generation(
            prompt="一颗红苹果",
            size="1024x1024",
            quality="high",
            output_format="png",
            output_dir=Path(temporary_directory) / "results",
            environ=environment,
            transport=transport,
            downloader=downloader,
        )
        return summary, Path(summary["record_dir"])

    def write_fake_key(self, temporary_directory: str) -> dict[str, str]:
        environment = {"CODEX_HOME": temporary_directory}
        auth_path = tool.auth_path(environment)
        auth_path.write_text(
            json.dumps({"OPENAI_API_KEY": FAKE_KEY}),
            encoding="utf-8",
        )
        return environment

    def assert_record_files(self, records: Path, expect_image: bool) -> None:
        required = {
            "request.json",
            "response.raw",
            "response-headers.json",
            "response.json",
            "summary.json",
        }
        self.assertTrue(required.issubset({path.name for path in records.iterdir()}))
        all_content = b"".join(path.read_bytes() for path in records.iterdir() if path.is_file())
        self.assertNotIn(FAKE_KEY.encode("utf-8"), all_content)
        if expect_image:
            self.assertEqual(len(list(records.glob("result.*"))), 1)


if __name__ == "__main__":
    unittest.main()
