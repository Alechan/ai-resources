#!/usr/bin/env python3
"""Run DataGrip eval fixtures and persist raw artifacts per fixture."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"Expected JSON object at {path}")
    return data


def dump_json(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        json.dump(data, handle, indent=2, sort_keys=True)
        handle.write("\n")


def validate_fixtures(data: dict[str, Any]) -> list[dict[str, Any]]:
    fixtures = data.get("fixtures")
    if not isinstance(fixtures, list) or not fixtures:
        raise ValueError("fixtures-v1.json must include a non-empty 'fixtures' list")

    seen_ids: set[str] = set()
    for fixture in fixtures:
        if not isinstance(fixture, dict):
            raise ValueError("Each fixture must be a JSON object")
        fixture_id = fixture.get("fixture_id")
        prompt_id = fixture.get("prompt_id")
        prompt_text = fixture.get("prompt_text")
        expected = fixture.get("expected")
        if not isinstance(fixture_id, str) or not fixture_id:
            raise ValueError("Each fixture requires non-empty string 'fixture_id'")
        if fixture_id in seen_ids:
            raise ValueError(f"Duplicate fixture_id: {fixture_id}")
        seen_ids.add(fixture_id)
        if not isinstance(prompt_id, str) or not prompt_id:
            raise ValueError(f"{fixture_id}: missing non-empty string 'prompt_id'")
        if not isinstance(prompt_text, str) or not prompt_text.strip():
            raise ValueError(f"{fixture_id}: missing non-empty string 'prompt_text'")
        if not isinstance(expected, dict):
            raise ValueError(f"{fixture_id}: missing object 'expected'")
    return fixtures


def normalize_response_text(stdout: str) -> str:
    stripped = stdout.strip()
    if not stripped:
        return ""
    try:
        parsed = json.loads(stripped)
    except json.JSONDecodeError:
        return stripped
    if isinstance(parsed, dict):
        candidate = parsed.get("response")
        if isinstance(candidate, str):
            return candidate.strip()
        candidate = parsed.get("output")
        if isinstance(candidate, str):
            return candidate.strip()
    return stripped


def run_fixture(
    fixture: dict[str, Any],
    fixture_dir: Path,
    dry_run: bool,
    runner_cmd: str | None,
    timeout_seconds: float,
) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "fixture_id": fixture["fixture_id"],
        "prompt_id": fixture["prompt_id"],
        "prompt_text": fixture["prompt_text"],
        "context": fixture.get("context", {}),
        "expected": fixture.get("expected", {}),
    }
    dump_json(fixture_dir / "input.json", payload)

    started = time.time()
    if dry_run:
        finished = time.time()
        result = {
            "status": "dry_run",
            "exit_code": 0,
            "duration_ms": int((finished - started) * 1000),
            "stdout": "",
            "stderr": "",
            "response_text": "",
            "error": None,
        }
        dump_json(fixture_dir / "result.json", result)
        return result

    if not runner_cmd:
        raise ValueError("runner command is required unless --dry-run is used")

    try:
        completed = subprocess.run(
            runner_cmd,
            input=json.dumps(payload),
            text=True,
            capture_output=True,
            shell=True,
            timeout=timeout_seconds,
            check=False,
        )
        finished = time.time()
        response_text = normalize_response_text(completed.stdout)
        result = {
            "status": "ok" if completed.returncode == 0 else "runner_error",
            "exit_code": completed.returncode,
            "duration_ms": int((finished - started) * 1000),
            "stdout": completed.stdout,
            "stderr": completed.stderr,
            "response_text": response_text,
            "error": None,
        }
    except subprocess.TimeoutExpired as exc:
        finished = time.time()
        result = {
            "status": "timeout",
            "exit_code": None,
            "duration_ms": int((finished - started) * 1000),
            "stdout": exc.stdout or "",
            "stderr": exc.stderr or "",
            "response_text": "",
            "error": f"runner timed out after {timeout_seconds:.1f}s",
        }

    dump_json(fixture_dir / "result.json", result)
    return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run DataGrip eval fixtures")
    parser.add_argument(
        "--fixtures",
        default="skills/datagrip-datasources/evals/fixtures-v1.json",
        help="Path to fixture JSON",
    )
    parser.add_argument(
        "--output-dir",
        default="skills/datagrip-datasources/evals/artifacts",
        help="Directory for run artifacts",
    )
    parser.add_argument(
        "--run-id",
        default="",
        help="Run identifier (defaults to UTC timestamp)",
    )
    parser.add_argument(
        "--runner-cmd",
        default="",
        help="Shell command to execute per fixture; receives fixture payload on stdin",
    )
    parser.add_argument(
        "--timeout-seconds",
        type=float,
        default=30.0,
        help="Timeout for each fixture execution",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Generate input/result artifacts without executing runner command",
    )
    parser.add_argument(
        "--fail-on-runner-error",
        action="store_true",
        help="Exit non-zero if any fixture returns runner error/timeout",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    fixtures_path = Path(args.fixtures)
    output_dir = Path(args.output_dir)
    if not fixtures_path.is_file():
        print(f"fixtures file not found: {fixtures_path}", file=sys.stderr)
        return 2

    try:
        fixtures_data = load_json(fixtures_path)
        fixtures = validate_fixtures(fixtures_data)
    except (ValueError, json.JSONDecodeError) as err:
        print(f"invalid fixtures file: {err}", file=sys.stderr)
        return 2

    if not args.dry_run and not args.runner_cmd.strip():
        print("--runner-cmd is required unless --dry-run is set", file=sys.stderr)
        return 2

    run_id = args.run_id.strip() or datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = output_dir / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    start_utc = datetime.now(timezone.utc).isoformat()
    summary_rows: list[dict[str, Any]] = []
    had_runner_error = False

    for fixture in fixtures:
        fixture_id = fixture["fixture_id"]
        fixture_dir = run_dir / fixture_id
        result = run_fixture(
            fixture=fixture,
            fixture_dir=fixture_dir,
            dry_run=args.dry_run,
            runner_cmd=args.runner_cmd.strip() or None,
            timeout_seconds=args.timeout_seconds,
        )
        status = str(result.get("status"))
        if status in {"runner_error", "timeout"}:
            had_runner_error = True

        summary_rows.append(
            {
                "fixture_id": fixture_id,
                "prompt_id": fixture["prompt_id"],
                "status": status,
                "exit_code": result.get("exit_code"),
                "duration_ms": result.get("duration_ms"),
                "artifact_dir": str((run_dir / fixture_id).as_posix()),
            }
        )

    end_utc = datetime.now(timezone.utc).isoformat()
    summary = {
        "suite_id": fixtures_data.get("suite_id"),
        "schema_version": fixtures_data.get("schema_version"),
        "run_id": run_id,
        "start_utc": start_utc,
        "end_utc": end_utc,
        "fixtures_total": len(summary_rows),
        "fixtures_ok": sum(1 for row in summary_rows if row["status"] == "ok"),
        "fixtures_dry_run": sum(1 for row in summary_rows if row["status"] == "dry_run"),
        "fixtures_runner_error": sum(1 for row in summary_rows if row["status"] == "runner_error"),
        "fixtures_timeout": sum(1 for row in summary_rows if row["status"] == "timeout"),
        "rows": summary_rows,
    }
    dump_json(run_dir / "run_summary.json", summary)

    print(f"Run artifacts written to: {run_dir.as_posix()}")
    print(
        "Summary:"
        f" total={summary['fixtures_total']}"
        f" ok={summary['fixtures_ok']}"
        f" dry_run={summary['fixtures_dry_run']}"
        f" runner_error={summary['fixtures_runner_error']}"
        f" timeout={summary['fixtures_timeout']}"
    )

    if args.fail_on_runner_error and had_runner_error:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
