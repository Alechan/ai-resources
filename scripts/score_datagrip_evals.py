#!/usr/bin/env python3
"""Score DataGrip eval run artifacts against assertion rules."""

from __future__ import annotations

import argparse
import json
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


def dump_markdown(path: Path, lines: list[str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        handle.write("\n".join(lines))
        handle.write("\n")


def normalize_text(value: str) -> str:
    return value.casefold()


def check_assertion(assertion: dict[str, Any], response_text: str) -> dict[str, Any]:
    conditions = assertion.get("pass_conditions", {})
    include_any = conditions.get("include_any", [])
    include_all = conditions.get("include_all", [])
    exclude_any = conditions.get("exclude_any", [])

    if not isinstance(include_any, list):
        include_any = []
    if not isinstance(include_all, list):
        include_all = []
    if not isinstance(exclude_any, list):
        exclude_any = []

    text = normalize_text(response_text)

    include_any_norm = [normalize_text(str(x)) for x in include_any]
    include_all_norm = [normalize_text(str(x)) for x in include_all]
    exclude_any_norm = [normalize_text(str(x)) for x in exclude_any]

    include_any_pass = True if not include_any_norm else any(p in text for p in include_any_norm)
    include_all_pass = all(p in text for p in include_all_norm)
    exclude_any_pass = all(p not in text for p in exclude_any_norm)
    passed = include_any_pass and include_all_pass and exclude_any_pass

    return {
        "assertion_id": assertion.get("assertion_id"),
        "priority": assertion.get("priority", "required"),
        "passed": passed,
        "checks": {
            "include_any_pass": include_any_pass,
            "include_all_pass": include_all_pass,
            "exclude_any_pass": exclude_any_pass,
        },
        "description": assertion.get("description", ""),
    }


def build_assertion_index(assertions_data: dict[str, Any]) -> dict[str, list[dict[str, Any]]]:
    prompts = assertions_data.get("prompts")
    if not isinstance(prompts, list):
        raise ValueError("assertions file must include 'prompts' list")
    index: dict[str, list[dict[str, Any]]] = {}
    for prompt in prompts:
        if not isinstance(prompt, dict):
            continue
        prompt_id = prompt.get("prompt_id")
        assertions = prompt.get("assertions")
        if isinstance(prompt_id, str) and isinstance(assertions, list):
            index[prompt_id] = assertions
    return index


def build_fixture_index(fixtures_data: dict[str, Any]) -> dict[str, dict[str, Any]]:
    fixtures = fixtures_data.get("fixtures")
    if not isinstance(fixtures, list):
        raise ValueError("fixtures file must include 'fixtures' list")
    index: dict[str, dict[str, Any]] = {}
    for fixture in fixtures:
        if not isinstance(fixture, dict):
            continue
        fixture_id = fixture.get("fixture_id")
        if isinstance(fixture_id, str):
            index[fixture_id] = fixture
    return index


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Score DataGrip eval run")
    parser.add_argument(
        "--assertions",
        default="skills/datagrip-datasources/evals/assertions-v1.json",
        help="Path to assertions JSON",
    )
    parser.add_argument(
        "--fixtures",
        default="skills/datagrip-datasources/evals/fixtures-v1.json",
        help="Path to fixtures JSON",
    )
    parser.add_argument(
        "--run-dir",
        required=True,
        help="Path to run artifacts directory (contains fixture dirs + run_summary.json)",
    )
    parser.add_argument(
        "--out-json",
        default="",
        help="Path for score summary JSON (default: <run-dir>/score_summary.json)",
    )
    parser.add_argument(
        "--out-md",
        default="",
        help="Path for score summary Markdown (default: <run-dir>/score_summary.md)",
    )
    parser.add_argument(
        "--fail-on-required",
        action="store_true",
        help="Exit non-zero if any required assertion fails",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    assertions_path = Path(args.assertions)
    fixtures_path = Path(args.fixtures)
    run_dir = Path(args.run_dir)
    out_json = Path(args.out_json) if args.out_json else run_dir / "score_summary.json"
    out_md = Path(args.out_md) if args.out_md else run_dir / "score_summary.md"

    if not assertions_path.is_file():
        raise SystemExit(f"assertions file not found: {assertions_path}")
    if not fixtures_path.is_file():
        raise SystemExit(f"fixtures file not found: {fixtures_path}")
    if not run_dir.is_dir():
        raise SystemExit(f"run directory not found: {run_dir}")

    assertions_data = load_json(assertions_path)
    fixtures_data = load_json(fixtures_path)
    assertion_index = build_assertion_index(assertions_data)
    fixture_index = build_fixture_index(fixtures_data)

    rows: list[dict[str, Any]] = []
    total_required = 0
    total_required_failed = 0
    total_advisory = 0
    total_advisory_failed = 0

    for fixture_id, fixture in sorted(fixture_index.items()):
        prompt_id = fixture.get("prompt_id")
        fixture_dir = run_dir / fixture_id
        result_path = fixture_dir / "result.json"

        if not result_path.is_file():
            rows.append(
                {
                    "fixture_id": fixture_id,
                    "prompt_id": prompt_id,
                    "status": "missing_result",
                    "required_failed": 1,
                    "advisory_failed": 0,
                    "assertions": [],
                    "errors": [f"missing {result_path.as_posix()}"],
                }
            )
            total_required += 1
            total_required_failed += 1
            continue

        result_data = load_json(result_path)
        response_text = str(result_data.get("response_text", ""))
        if not response_text:
            response_text = str(result_data.get("stdout", ""))

        prompt_assertions = assertion_index.get(str(prompt_id), [])
        if not prompt_assertions:
            rows.append(
                {
                    "fixture_id": fixture_id,
                    "prompt_id": prompt_id,
                    "status": result_data.get("status"),
                    "required_failed": 1,
                    "advisory_failed": 0,
                    "assertions": [],
                    "errors": [f"no assertions configured for prompt_id={prompt_id}"],
                }
            )
            total_required += 1
            total_required_failed += 1
            continue
        assertion_results = [check_assertion(a, response_text) for a in prompt_assertions]

        required_failed = 0
        advisory_failed = 0
        for item in assertion_results:
            if item["priority"] == "required":
                total_required += 1
                if not item["passed"]:
                    required_failed += 1
                    total_required_failed += 1
            else:
                total_advisory += 1
                if not item["passed"]:
                    advisory_failed += 1
                    total_advisory_failed += 1

        rows.append(
            {
                "fixture_id": fixture_id,
                "prompt_id": prompt_id,
                "status": result_data.get("status"),
                "required_failed": required_failed,
                "advisory_failed": advisory_failed,
                "assertions": assertion_results,
                "errors": [],
            }
        )

    summary = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "suite_id": assertions_data.get("suite_id"),
        "run_dir": run_dir.as_posix(),
        "fixtures_total": len(rows),
        "fixtures_with_required_failures": sum(1 for row in rows if row["required_failed"] > 0),
        "required_total": total_required,
        "required_failed": total_required_failed,
        "advisory_total": total_advisory,
        "advisory_failed": total_advisory_failed,
        "rows": rows,
    }
    dump_json(out_json, summary)

    md_lines = [
        "# DataGrip Eval Score Summary",
        "",
        f"- Generated at: {summary['generated_at']}",
        f"- Run dir: `{summary['run_dir']}`",
        f"- Fixtures total: {summary['fixtures_total']}",
        f"- Fixtures with required failures: {summary['fixtures_with_required_failures']}",
        f"- Required assertions: {summary['required_total']} (failed: {summary['required_failed']})",
        f"- Advisory assertions: {summary['advisory_total']} (failed: {summary['advisory_failed']})",
        "",
        "## Per-fixture",
        "",
    ]

    for row in rows:
        md_lines.append(
            f"- `{row['fixture_id']}` ({row['prompt_id']}): status={row['status']}, "
            f"required_failed={row['required_failed']}, advisory_failed={row['advisory_failed']}"
        )
    dump_markdown(out_md, md_lines)

    print(f"Wrote score JSON: {out_json.as_posix()}")
    print(f"Wrote score Markdown: {out_md.as_posix()}")
    print(
        "Score summary:"
        f" fixtures={summary['fixtures_total']}"
        f" required_failed={summary['required_failed']}"
        f" advisory_failed={summary['advisory_failed']}"
    )

    if args.fail_on_required and summary["required_failed"] > 0:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
