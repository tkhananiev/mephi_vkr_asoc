#!/usr/bin/env python3
"""Читает Semgrep JSON (--json), пишет тело POST /api/v1/findings/ingest в stdout."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path


def map_severity(raw: str) -> str:
    s = (raw or "").strip().upper()
    if s == "ERROR":
        return "high"
    if s == "WARNING":
        return "medium"
    if s == "INFO":
        return "low"
    return "unknown"


def cwe_from_meta(meta: dict) -> str:
    if not isinstance(meta, dict):
        return ""
    cwe = meta.get("cwe")
    if isinstance(cwe, list) and cwe:
        first = cwe[0]
        if isinstance(first, str):
            return first.strip()
        if isinstance(first, dict):
            return str(first.get("id", "") or first.get("name", "") or "").strip()
    if isinstance(cwe, str):
        return cwe.strip()
    return ""


def cve_from_meta(meta: dict) -> str:
    if not isinstance(meta, dict):
        return ""
    cve = meta.get("cve")
    if isinstance(cve, list) and cve:
        x = cve[0]
        return x.strip() if isinstance(x, str) else ""
    if isinstance(cve, str):
        return cve.strip()
    return ""


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
        print("usage: semgrep_to_asoc_payload.py <semgrep-report.json>", file=sys.stderr)
        return 2
    path = Path(sys.argv[1])
    if not path.is_file():
        body = {
            "scanner_name": "semgrep-ci",
            "channel": "ci",
            "findings": [],
        }
        attach_console_product_root(body)
        print(json.dumps(body, ensure_ascii=False))
        return 0

    data = json.loads(path.read_text(encoding="utf-8"))
    results = data.get("results") if isinstance(data, dict) else None
    if not isinstance(results, list):
        results = []

    commit = os.environ.get("CI_COMMIT_SHORT_SHA", "unknown").strip() or "unknown"
    proj = os.environ.get("CI_PROJECT_PATH", "ci").strip() or "ci"

    findings = []
    for r in results:
        if not isinstance(r, dict):
            continue
        p = str(r.get("path") or "").strip()
        check_id = str(r.get("check_id") or "semgrep").strip()
        extra = r.get("extra") if isinstance(r.get("extra"), dict) else {}
        meta = extra.get("metadata") if isinstance(extra.get("metadata"), dict) else {}
        sev = map_severity(str(extra.get("severity") or ""))
        start = r.get("start") if isinstance(r.get("start"), dict) else {}
        line = start.get("line")
        line_s = str(line) if line is not None else "0"

        asset_id = f"{proj}:{commit}:{p}:{line_s}"
        identifier = check_id if len(check_id) <= 512 else check_id[:509] + "..."
        msg = str(extra.get("message") or "").strip()

        findings.append(
            {
                "asset_id": asset_id[:2048],
                "identifier": identifier[:2048],
                "severity": sev,
                "component": p[:512] or "repository",
                "version": commit[:128],
                "cve": cve_from_meta(meta)[:128],
                "cwe": cwe_from_meta(meta)[:128],
                "metadata": merge_console_product_meta(
                    {"gitlab_ci": True, "scanner": "semgrep", "message": msg[:2000]}
                ),
                "raw_payload": r,
            }
        )

    body = {"scanner_name": "semgrep-ci", "channel": "ci", "findings": findings}
    attach_console_product_root(body)
    print(json.dumps(body, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
