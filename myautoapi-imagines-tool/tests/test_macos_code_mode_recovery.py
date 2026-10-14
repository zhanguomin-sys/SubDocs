"""Linux simulations for the macOS Code Mode image-result fallback."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import tempfile
import time
import unittest
from dataclasses import dataclass
from pathlib import Path


TINY_PNG = bytes.fromhex(
    "89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c489"
    "0000000d4944415408d763f8cfc0f01f00050001ff89993d1d0000000049454e44ae426082"
)
IMAGE_EXTENSIONS = {".png", ".jpg", ".jpeg", ".webp"}


@dataclass(frozen=True)
class RecoveryResult:
    status: str
    paths: tuple[Path, ...]


def sanitize_output_name(value: str) -> str:
    """Mirror the filename characters relevant to recovery matching."""

    stripped = value.strip()
    if Path(stripped).suffix.lower() in IMAGE_EXTENSIONS:
        stripped = stripped[: -len(Path(stripped).suffix)]
    output: list[str] = []
    last_was_separator = False
    for character in stripped:
        invalid = ord(character) < 32 or character in '<>:"/\\|?*'
        if invalid:
            if not last_was_separator:
                output.append("-")
                last_was_separator = True
            continue
        output.append(character)
        last_was_separator = False
    cleaned = "".join(output).strip(" .-")
    return cleaned or "图片"


def recover_candidates(
    output_dir: Path,
    output_name: str,
    expected_count: int,
    started_ns: int,
) -> RecoveryResult:
    """Reference implementation of the bounded fallback described by the Skill."""

    escaped_name = re.escape(sanitize_output_name(output_name))
    if expected_count == 1:
        name_pattern = re.compile(rf"^{escaped_name}(?:-[0-9]+)?$")
    else:
        indexes = "|".join(str(index) for index in range(1, expected_count + 1))
        name_pattern = re.compile(rf"^{escaped_name}-(?:{indexes})(?:-[0-9]+)?$")

    candidates: list[Path] = []
    for path in output_dir.iterdir():
        if not path.is_file() or path.suffix.lower() not in IMAGE_EXTENSIONS:
            continue
        if path.stat().st_mtime_ns < started_ns:
            continue
        if not name_pattern.fullmatch(path.stem):
            continue
        candidates.append(path)
    candidates.sort(key=lambda path: (path.stat().st_mtime_ns, path.name))

    if len(candidates) == expected_count:
        return RecoveryResult("complete", tuple(candidates))
    if len(candidates) < expected_count:
        return RecoveryResult("partial", tuple(candidates))
    return RecoveryResult("ambiguous", ())


def run_fake_tool(output_dir: Path, output_name: str, count: int = 1) -> subprocess.CompletedProcess[str]:
    """Write valid PNG files and emit the same success JSON shape as the real tool."""

    fake_tool = """
import json
import sys
from pathlib import Path

output_dir = Path(sys.argv[1])
output_name = sys.argv[2]
count = int(sys.argv[3])
payload = bytes.fromhex(sys.argv[4])
output_dir.mkdir(parents=True, exist_ok=True)
paths = []
for index in range(1, count + 1):
    suffix = '' if count == 1 else f'-{index}'
    path = output_dir / f'{output_name}{suffix}.png'
    path.write_bytes(payload)
    paths.append(str(path))
summary = {'ok': True}
summary['output_file' if count == 1 else 'output_files'] = paths[0] if count == 1 else paths
print(json.dumps(summary, ensure_ascii=False))
"""
    return subprocess.run(
        [
            sys.executable,
            "-c",
            fake_tool,
            str(output_dir),
            output_name,
            str(count),
            TINY_PNG.hex(),
        ],
        text=True,
        capture_output=True,
        check=False,
    )


class MacOSCodeModeRecoverySimulationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.output_dir = Path(self.temporary_directory.name) / "images"
        self.output_dir.mkdir()

    def write_image(self, name: str, modified_ns: int) -> Path:
        path = self.output_dir / name
        path.write_bytes(TINY_PNG)
        os.utime(path, ns=(modified_ns, modified_ns))
        return path

    def test_explicit_forwarding_preserves_success_json(self) -> None:
        completed = run_fake_tool(self.output_dir, "白底红苹果")
        self.assertEqual(completed.returncode, 0)
        self.assertEqual(completed.stderr, "")

        nested_result = {
            "exit_code": completed.returncode,
            "output": completed.stdout,
            "session_id": None,
        }
        outer_text = json.dumps(nested_result, ensure_ascii=False)
        forwarded = json.loads(outer_text)
        summary = json.loads(forwarded["output"])

        self.assertTrue(summary["ok"])
        self.assertTrue(Path(summary["output_file"]).is_file())

    def test_swallowed_outer_result_recovers_single_written_image(self) -> None:
        started_ns = time.time_ns()
        completed = run_fake_tool(self.output_dir, "白底红苹果")
        outer_text = ""  # Simulates a wrapper that never calls text(...).

        self.assertEqual(completed.returncode, 0)
        self.assertEqual(outer_text, "")
        recovered = recover_candidates(self.output_dir, "白底红苹果", 1, started_ns)
        self.assertEqual(recovered.status, "complete")
        self.assertEqual([path.name for path in recovered.paths], ["白底红苹果.png"])

    def test_recovery_accepts_collision_suffix_and_excludes_old_or_unrelated_images(self) -> None:
        old_ns = time.time_ns() - 10_000_000_000
        started_ns = time.time_ns()
        new_ns = started_ns + 1_000_000
        self.write_image("白底红苹果.png", old_ns)
        expected = self.write_image("白底红苹果-2.png", new_ns)
        self.write_image("无关新图.png", new_ns + 1)

        recovered = recover_candidates(self.output_dir, "白底红苹果", 1, started_ns)
        self.assertEqual(recovered, RecoveryResult("complete", (expected,)))

    def test_recovery_matches_the_tools_sanitized_output_name(self) -> None:
        started_ns = time.time_ns()
        expected = self.write_image("海报-春季.png", started_ns + 1)

        recovered = recover_candidates(self.output_dir, "海报/春季", 1, started_ns)
        self.assertEqual(recovered, RecoveryResult("complete", (expected,)))

    def test_recovery_strips_a_known_extension_from_output_name(self) -> None:
        started_ns = time.time_ns()
        expected = self.write_image("海报.png", started_ns + 1)

        recovered = recover_candidates(self.output_dir, "海报.png", 1, started_ns)
        self.assertEqual(recovered, RecoveryResult("complete", (expected,)))

    def test_recovery_returns_multi_image_results_in_stable_order(self) -> None:
        started_ns = time.time_ns()
        later_ns = started_ns + 2_000_000
        earlier_ns = started_ns + 1_000_000
        second = self.write_image("海报-2.png", later_ns)
        first = self.write_image("海报-1.png", earlier_ns)
        self.write_image("别的海报.png", later_ns + 1)

        recovered = recover_candidates(self.output_dir, "海报", 2, started_ns)
        self.assertEqual(recovered.status, "complete")
        self.assertEqual(recovered.paths, (first, second))

    def test_partial_result_is_reported_without_claiming_complete_success(self) -> None:
        started_ns = time.time_ns()
        first = self.write_image("海报-1.png", started_ns + 1)

        recovered = recover_candidates(self.output_dir, "海报", 3, started_ns)
        self.assertEqual(recovered, RecoveryResult("partial", (first,)))

    def test_too_many_matching_results_are_ambiguous(self) -> None:
        started_ns = time.time_ns()
        self.write_image("海报.png", started_ns + 1)
        self.write_image("海报-2.png", started_ns + 2)

        recovered = recover_candidates(self.output_dir, "海报", 1, started_ns)
        self.assertEqual(recovered, RecoveryResult("ambiguous", ()))

    def test_same_prefix_parallel_results_cannot_be_bound_to_models(self) -> None:
        started_ns = time.time_ns()
        self.write_image("海报.png", started_ns + 1)
        self.write_image("海报-2.png", started_ns + 2)

        first_call = recover_candidates(self.output_dir, "海报", 1, started_ns)
        second_call = recover_candidates(self.output_dir, "海报", 1, started_ns)

        self.assertEqual(first_call.status, "ambiguous")
        self.assertEqual(second_call.status, "ambiguous")
        self.assertEqual(first_call.paths, ())
        self.assertEqual(second_call.paths, ())


if __name__ == "__main__":
    unittest.main()
