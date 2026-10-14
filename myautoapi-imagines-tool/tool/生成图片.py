#!/usr/bin/env python3
"""Generate one image with MyAutoAPI's gpt-image-2 endpoint.

V0.2 deliberately has one supported model and one fixed API endpoint. The API
key is read by this script from Codex's auth.json; callers never pass it on the
command line.
"""

from __future__ import annotations

import argparse
import base64
import binascii
import json
import os
import sys
from collections.abc import Callable, Mapping
from datetime import datetime
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlparse
from urllib.request import Request, urlopen


API_URL = "https://api.myautoapi.com/v1/images/generations"
MODEL = "gpt-image-2"
TOOL_DIRECTORY = "myautoapi-imagines-tool"
AUTH_FILE_NAME = "auth.json"
AUTH_KEY_FIELD = "OPENAI_API_KEY"
DEFAULT_OUTPUT_DIR_NAME = "images"
DEFAULT_SIZE = "1024x1024"
DEFAULT_QUALITY = "high"
DEFAULT_OUTPUT_FORMAT = "png"
DEFAULT_TIMEOUT_SECONDS = 180


class ToolError(Exception):
    """Expected, non-sensitive tool failure."""


class InputError(ToolError):
    """An unsupported or incomplete caller input."""


class GenerationError(ToolError):
    """A failed request with a directory containing its available evidence."""

    def __init__(self, message: str, record_dir: Path | None = None) -> None:
        super().__init__(message)
        self.record_dir = record_dir


class HttpResponse:
    """Small injectable HTTP response type for the standard-library transport."""

    def __init__(self, status: int, headers: Mapping[str, str] | None, body: bytes) -> None:
        self.status = int(status)
        self.headers = dict(headers or {})
        self.body = body


Transport = Callable[[bytes, str], HttpResponse]
Downloader = Callable[[str], HttpResponse]


def codex_home(environ: Mapping[str, str] | None = None) -> Path:
    environment = os.environ if environ is None else environ
    configured_home = environment.get("CODEX_HOME", "").strip()
    if configured_home:
        return Path(configured_home).expanduser()
    return Path.home() / ".codex"


def auth_path(environ: Mapping[str, str] | None = None) -> Path:
    return codex_home(environ) / AUTH_FILE_NAME


def default_output_dir(environ: Mapping[str, str] | None = None) -> Path:
    return codex_home(environ) / DEFAULT_OUTPUT_DIR_NAME


def load_api_key(environ: Mapping[str, str] | None = None) -> str:
    path = auth_path(environ)
    if path.is_symlink() or (path.exists() and not path.is_file()):
        raise ToolError(f"Codex auth.json 不是普通文件：{path}")
    try:
        content = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as error:
        raise ToolError(f"无法读取 Codex auth.json：{path}") from error
    if not isinstance(content, dict):
        raise ToolError(f"Codex auth.json 格式无效：{path}")
    key = content.get(AUTH_KEY_FIELD)
    if not isinstance(key, str) or not key.strip():
        raise ToolError(f"Codex auth.json 中缺少可用的 {AUTH_KEY_FIELD}。")
    return key.strip()


def validate_model(model: str) -> str:
    if model != MODEL:
        raise InputError(f"V0.1 仅支持模型：{MODEL}")
    return model


def require_non_empty(name: str, value: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise InputError(f"{name} 不能为空。")
    return value.strip()


def build_payload(prompt: str, size: str, quality: str, output_format: str) -> dict[str, object]:
    return {
        "model": MODEL,
        "prompt": require_non_empty("图片描述", prompt),
        "n": 1,
        "size": require_non_empty("size", size),
        "quality": require_non_empty("quality", quality),
        "output_format": require_non_empty("output_format", output_format),
    }


def _json_text(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n"


def _redact_text(value: str, secrets: tuple[str, ...]) -> str:
    redacted = value
    for secret in secrets:
        if secret:
            redacted = redacted.replace(secret, "[REDACTED]")
    return redacted


def _redact_bytes(value: bytes, secrets: tuple[str, ...]) -> bytes:
    redacted = value
    for secret in secrets:
        if secret:
            redacted = redacted.replace(secret.encode("utf-8"), b"[REDACTED]")
    return redacted


def write_json(path: Path, value: object, secrets: tuple[str, ...] = ()) -> None:
    path.write_text(_redact_text(_json_text(value), secrets), encoding="utf-8")


def create_record_dir(output_dir: Path) -> Path:
    timestamp = datetime.now().astimezone().strftime("%Y%m%d-%H%M%S-%f")
    record_dir = output_dir / f"{timestamp}-{MODEL}"
    record_dir.mkdir(parents=True, exist_ok=False)
    return record_dir


def default_transport(request_body: bytes, api_key: str) -> HttpResponse:
    request = Request(
        API_URL,
        data=request_body,
        method="POST",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "User-Agent": "myautoapi-imagines-tool/0.1",
        },
    )
    try:
        with urlopen(request, timeout=DEFAULT_TIMEOUT_SECONDS) as response:
            return HttpResponse(response.status, dict(response.headers.items()), response.read())
    except HTTPError as error:
        headers = dict(error.headers.items()) if error.headers else {}
        return HttpResponse(error.code, headers, error.read())
    except URLError as error:
        raise ToolError(f"MyAutoAPI 请求无法发送：{error.reason}") from error


def _is_https_url(value: object) -> bool:
    if not isinstance(value, str):
        return False
    parsed = urlparse(value)
    return parsed.scheme == "https" and bool(parsed.netloc)


def default_downloader(url: str) -> HttpResponse:
    request = Request(url, headers={"User-Agent": "myautoapi-imagines-tool/0.1"})
    try:
        with urlopen(request, timeout=DEFAULT_TIMEOUT_SECONDS) as response:
            return HttpResponse(response.status, dict(response.headers.items()), response.read())
    except HTTPError as error:
        headers = dict(error.headers.items()) if error.headers else {}
        return HttpResponse(error.code, headers, error.read())
    except URLError as error:
        raise ToolError(f"图片下载失败：{error.reason}") from error


def png_dimensions(image_bytes: bytes) -> tuple[int, int] | None:
    if len(image_bytes) < 24 or not image_bytes.startswith(b"\x89PNG\r\n\x1a\n"):
        return None
    return (
        int.from_bytes(image_bytes[16:20], "big"),
        int.from_bytes(image_bytes[20:24], "big"),
    )


def jpeg_dimensions(image_bytes: bytes) -> tuple[int, int] | None:
    if len(image_bytes) < 4 or not image_bytes.startswith(b"\xff\xd8"):
        return None
    position = 2
    start_of_frame_markers = {
        0xC0,
        0xC1,
        0xC2,
        0xC3,
        0xC5,
        0xC6,
        0xC7,
        0xC9,
        0xCA,
        0xCB,
        0xCD,
        0xCE,
        0xCF,
    }
    while position + 4 <= len(image_bytes):
        if image_bytes[position] != 0xFF:
            position += 1
            continue
        while position < len(image_bytes) and image_bytes[position] == 0xFF:
            position += 1
        if position >= len(image_bytes):
            break
        marker = image_bytes[position]
        position += 1
        if marker in {0xD8, 0xD9} or 0xD0 <= marker <= 0xD7:
            continue
        if position + 2 > len(image_bytes):
            break
        segment_length = int.from_bytes(image_bytes[position : position + 2], "big")
        if segment_length < 2 or position + segment_length > len(image_bytes):
            break
        if marker in start_of_frame_markers and segment_length >= 7:
            height = int.from_bytes(image_bytes[position + 3 : position + 5], "big")
            width = int.from_bytes(image_bytes[position + 5 : position + 7], "big")
            return width, height
        position += segment_length
    return None


def webp_dimensions(image_bytes: bytes) -> tuple[int, int] | None:
    if len(image_bytes) < 30 or not image_bytes.startswith(b"RIFF") or image_bytes[8:12] != b"WEBP":
        return None
    if image_bytes[12:16] != b"VP8X":
        return None
    width = int.from_bytes(image_bytes[24:27], "little") + 1
    height = int.from_bytes(image_bytes[27:30], "little") + 1
    return width, height


def inspect_image(image_bytes: bytes) -> tuple[str, str, tuple[int, int] | None]:
    dimensions = png_dimensions(image_bytes)
    if dimensions is not None:
        return "png", "png", dimensions
    dimensions = jpeg_dimensions(image_bytes)
    if dimensions is not None:
        return "jpeg", "jpg", dimensions
    if len(image_bytes) >= 10 and image_bytes[:6] in {b"GIF87a", b"GIF89a"}:
        return (
            "gif",
            "gif",
            (int.from_bytes(image_bytes[6:8], "little"), int.from_bytes(image_bytes[8:10], "little")),
        )
    dimensions = webp_dimensions(image_bytes)
    if dimensions is not None or (
        len(image_bytes) >= 12 and image_bytes.startswith(b"RIFF") and image_bytes[8:12] == b"WEBP"
    ):
        return "webp", "webp", dimensions
    return "unknown", "bin", None


def decode_image_base64(value: object) -> bytes | None:
    if not isinstance(value, str):
        return None
    candidate = value.strip()
    if candidate.startswith("data:") and "," in candidate:
        candidate = candidate.split(",", 1)[1]
    if not candidate:
        return None
    try:
        decoded = base64.b64decode(candidate, validate=True)
    except (ValueError, binascii.Error):
        return None
    if not decoded or inspect_image(decoded)[0] == "unknown":
        return None
    return decoded


def extract_image(
    response_json: object, downloader: Downloader
) -> tuple[bytes, str, Mapping[str, str] | None]:
    if not isinstance(response_json, Mapping):
        raise ToolError("图片接口返回的 JSON 结构无效。")
    data = response_json.get("data")
    if not isinstance(data, list):
        raise ToolError("图片接口返回中没有 data 列表。")

    for item in data:
        if isinstance(item, Mapping):
            image_bytes = decode_image_base64(item.get("b64_json"))
            if image_bytes is not None:
                return image_bytes, "b64_json", None

    for item in data:
        if isinstance(item, Mapping) and _is_https_url(item.get("url")):
            image_response = downloader(str(item["url"]))
            if not 200 <= image_response.status < 300:
                raise ToolError(f"图片 URL 下载返回 HTTP {image_response.status}。")
            if not image_response.body:
                raise ToolError("图片 URL 下载结果为空。")
            return image_response.body, "url", image_response.headers

    raise ToolError("图片接口未返回可保存的 b64_json 或 HTTPS URL。")


def _decode_response_json(response_body: bytes) -> object | None:
    try:
        return json.loads(response_body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return None


def _failure_summary(record_dir: Path, message: str, status: int | None = None) -> dict[str, object]:
    summary: dict[str, object] = {
        "ok": False,
        "model": MODEL,
        "record_dir": str(record_dir),
        "error": message,
    }
    if status is not None:
        summary["status"] = status
    return summary


def _write_failure(record_dir: Path, message: str, api_key: str = "", status: int | None = None) -> None:
    write_json(record_dir / "summary.json", _failure_summary(record_dir, message, status), (api_key,))


def run_generation(
    *,
    prompt: str,
    size: str = DEFAULT_SIZE,
    quality: str = DEFAULT_QUALITY,
    output_format: str = DEFAULT_OUTPUT_FORMAT,
    output_dir: Path | str | None = None,
    model: str = MODEL,
    environ: Mapping[str, str] | None = None,
    transport: Transport | None = None,
    downloader: Downloader | None = None,
) -> dict[str, object]:
    validate_model(model)
    payload = build_payload(prompt, size, quality, output_format)
    record_dir = create_record_dir(
        default_output_dir(environ) if output_dir is None else Path(output_dir)
    )
    request_body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    write_json(record_dir / "request.json", {"endpoint": API_URL, "payload": payload})

    try:
        api_key = load_api_key(environ)
    except ToolError as error:
        _write_failure(record_dir, str(error))
        raise GenerationError(str(error), record_dir) from error

    request_transport = default_transport if transport is None else transport
    image_downloader = default_downloader if downloader is None else downloader
    try:
        response = request_transport(request_body, api_key)
    except ToolError as error:
        _write_failure(record_dir, str(error), api_key)
        raise GenerationError(str(error), record_dir) from error
    except OSError as error:
        message = f"MyAutoAPI 请求失败：{error}"
        _write_failure(record_dir, message, api_key)
        raise GenerationError(message, record_dir) from error

    response_body = response.body
    safe_response_body = _redact_bytes(response_body, (api_key,))
    (record_dir / "response.raw").write_bytes(safe_response_body)
    write_json(record_dir / "response-headers.json", response.headers, (api_key,))
    response_json = _decode_response_json(response_body)
    if response_json is not None:
        write_json(record_dir / "response.json", response_json, (api_key,))

    if not 200 <= response.status < 300:
        message = f"MyAutoAPI 返回 HTTP {response.status}。"
        _write_failure(record_dir, message, api_key, response.status)
        raise GenerationError(message, record_dir)
    if response_json is None:
        message = "MyAutoAPI 成功响应不是可解析的 JSON。"
        _write_failure(record_dir, message, api_key, response.status)
        raise GenerationError(message, record_dir)

    try:
        image_bytes, source, image_headers = extract_image(response_json, image_downloader)
    except ToolError as error:
        _write_failure(record_dir, str(error), api_key, response.status)
        raise GenerationError(str(error), record_dir) from error

    if image_headers is not None:
        write_json(record_dir / "image-headers.json", image_headers, (api_key,))
    image_format, extension, dimensions = inspect_image(image_bytes)
    output_file = record_dir / f"result.{extension}"
    output_file.write_bytes(image_bytes)
    summary: dict[str, object] = {
        "ok": True,
        "model": MODEL,
        "source": source,
        "format": image_format,
        "bytes": len(image_bytes),
        "dimensions": list(dimensions) if dimensions is not None else None,
        "record_dir": str(record_dir),
        "output_file": str(output_file),
        "status": response.status,
    }
    write_json(record_dir / "summary.json", summary, (api_key,))
    return summary


def create_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="使用 MyAutoAPI 的 gpt-image-2 生成一张图片。")
    parser.add_argument("--prompt", required=True, help="图片描述。")
    parser.add_argument("--model", default=MODEL, help=f"V0.1 固定为 {MODEL}。")
    parser.add_argument("--size", default=DEFAULT_SIZE, help=f"图片尺寸，默认 {DEFAULT_SIZE}。")
    parser.add_argument("--quality", default=DEFAULT_QUALITY, help=f"图片质量，默认 {DEFAULT_QUALITY}。")
    parser.add_argument(
        "--output-format",
        default=DEFAULT_OUTPUT_FORMAT,
        help=f"请求输出格式，默认 {DEFAULT_OUTPUT_FORMAT}。",
    )
    parser.add_argument(
        "--output-dir",
        default=None,
        help="结果根目录，默认 CODEX_HOME 下的 images。",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    arguments = create_parser().parse_args(argv)
    try:
        summary = run_generation(
            prompt=arguments.prompt,
            model=arguments.model,
            size=arguments.size,
            quality=arguments.quality,
            output_format=arguments.output_format,
            output_dir=Path(arguments.output_dir) if arguments.output_dir else None,
        )
    except GenerationError as error:
        failure = {"ok": False, "error": str(error)}
        if error.record_dir is not None:
            failure["record_dir"] = str(error.record_dir)
        print(json.dumps(failure, ensure_ascii=False), file=sys.stderr)
        return 1
    except ToolError as error:
        print(json.dumps({"ok": False, "error": str(error)}, ensure_ascii=False), file=sys.stderr)
        return 1
    except OSError as error:
        print(json.dumps({"ok": False, "error": f"本地文件操作失败：{error}"}, ensure_ascii=False), file=sys.stderr)
        return 1
    print(json.dumps(summary, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
