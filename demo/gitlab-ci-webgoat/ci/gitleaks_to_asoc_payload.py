#!/usr/bin/env python3
"""Читает JSON-отчёт Gitleaks (--report-format json), пишет тело POST /api/v1/findings/ingest в stdout."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path


def merge_console_product_meta(metadata: dict) -> dict:
    pid = os.environ.get("ASOC_CONSOLE_PRODUCT_ID", "").strip()
    if pid:
        return {**metadata, "console_product_id": pid}
    return metadata


def attach_console_product_root(body: dict) -> None:
    raw = os.environ.get("ASOC_CONSOLE_PRODUCT_ID", "").strip()
    if not raw:
        return
    try:
        body["console_product_id"] = int(raw)
    except ValueError:
        pass


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: gitleaks_to_asoc_payload.py <gitleaks-report.json>", file=sys.stderr)
        return 2
    path = Path(sys.argv[1])
    if not path.is_file():
        body = {"scanner_name": "gitleaks-ci", "channel": "ci", "findings": []}
        attach_console_product_root(body)
        print(json.dumps(body, ensure_ascii=False))
        return 0

    raw = json.loads(path.read_text(encoding="utf-8"))
    # Gitleaks v8: массив находок или объект с ключом в зависимости от версии
    rows = raw if isinstance(raw, list) else raw.get("findings") or raw.get("leaks") or []
    if not isinstance(rows, list):
        rows = []

    commit = os.environ.get("CI_COMMIT_SHORT_SHA", "unknown").strip() or "unknown"
    proj = os.environ.get("CI_PROJECT_PATH", "ci").strip() or "ci"

    findings = []
    for row in rows:
        if not isinstance(row, dict):
            continue
        rule = str(row.get("RuleID") or row.get("rule") or "gitleaks").strip()
        fpath = str(row.get("File") or row.get("file") or "").strip()
        line = row.get("StartLine") or row.get("startLine") or row.get("line") or 0
        desc = str(row.get("Description") or row.get("description") or "").strip()

        identifier = f"{rule}:{fpath}:{line}"
        asset_id = f"{proj}:{commit}:{fpath}:{line}"

        findings.append(
            {
                "asset_id": asset_id[:2048],
                "identifier": identifier[:2048],
                "severity": "high",
                "component": fpath[:512] or "repository",
                "version": commit[:128],
                "cve": "",
                "cwe": "",
                "metadata": merge_console_product_meta(
                    {"gitlab_ci": True, "scanner": "gitleaks", "description": desc[:2000]}
                ),
                "raw_payload": row,
            }
        )

    body = {"scanner_name": "gitleaks-ci", "channel": "ci", "findings": findings}
    attach_console_product_root(body)
    print(json.dumps(body, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
