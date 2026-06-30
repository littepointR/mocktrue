#!/usr/bin/env python3
"""Validate the Qt/Rust migration feature matrix.

This is intentionally conservative: it catches missing rows, malformed CSV,
bad statuses, and first-wave rows that are marked contracted without enough
implementation/test anchors.
"""

from __future__ import annotations

import csv
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MATRIX = ROOT / "docs" / "development" / "migration-feature-matrix.csv"
REQUIRED_COLUMNS = [
    "feature_id",
    "domain",
    "old_ui",
    "old_frontend_state",
    "old_binding",
    "old_backend",
    "old_mcp_tool",
    "old_docs",
    "old_tests",
    "new_rust_api",
    "new_qt_viewmodel",
    "new_qml_entry",
    "new_tests",
    "status",
    "owner",
    "acceptance",
    "notes",
]
VALID_STATUS = {
    "inventory",
    "contracted",
    "implemented",
    "tested",
    "accepted",
    "removed",
    "deferred",
}
VALID_DOMAINS = {
    "shell",
    "workspace",
    "settings",
    "serial",
    "virtual",
    "graph",
    "protocol",
    "mcp",
    "runtime",
    "release",
}


def read_matrix() -> list[dict[str, str]]:
    with MATRIX.open(newline="", encoding="utf-8") as fh:
        reader = csv.DictReader(fh)
        if reader.fieldnames != REQUIRED_COLUMNS:
            raise SystemExit(
                "migration matrix columns do not match expected schema:\n"
                f"expected: {REQUIRED_COLUMNS}\n"
                f"actual:   {reader.fieldnames}"
            )
        return list(reader)


def exported_binding_names() -> set[str]:
    names: set[str] = set()
    for path in (ROOT / "frontend" / "bindings").rglob("service.ts"):
        text = path.read_text(encoding="utf-8")
        names.update(re.findall(r"^export function ([A-Za-z0-9_]+)\(", text, re.M))
    return names


def mcp_tool_names() -> set[str]:
    path = ROOT / "internal" / "modules" / "mcpserver" / "server_test.go"
    text = path.read_text(encoding="utf-8")
    return set(re.findall(r'"((?:serial|modbus|fecbus|protocol)_[a-z0-9_]+)"', text))


def main() -> int:
    rows = read_matrix()
    errors: list[str] = []
    by_feature: dict[str, dict[str, str]] = {}

    for idx, row in enumerate(rows, start=2):
        feature_id = row["feature_id"].strip()
        if not feature_id:
            errors.append(f"line {idx}: missing feature_id")
            continue
        if feature_id in by_feature:
            errors.append(f"line {idx}: duplicate feature_id {feature_id}")
        by_feature[feature_id] = row

        if row["status"] not in VALID_STATUS:
            errors.append(f"line {idx}: invalid status {row['status']!r}")
        if row["domain"] not in VALID_DOMAINS:
            errors.append(f"line {idx}: invalid domain {row['domain']!r}")
        if not row["owner"]:
            errors.append(f"line {idx}: missing owner for {feature_id}")
        if not row["acceptance"]:
            errors.append(f"line {idx}: missing acceptance for {feature_id}")

        if row["status"] in {"contracted", "implemented", "tested", "accepted"}:
            for col in ("new_rust_api", "new_qt_viewmodel", "new_qml_entry", "new_tests"):
                if not row[col].strip():
                    errors.append(f"line {idx}: {feature_id} is {row['status']} but missing {col}")

        if row["status"] == "accepted" and not row["new_tests"].strip():
            errors.append(f"line {idx}: accepted row lacks tests: {feature_id}")
        if row["status"] in {"removed", "deferred"} and not row["notes"].strip():
            errors.append(f"line {idx}: {row['status']} row lacks notes: {feature_id}")

    binding_names = exported_binding_names()
    matrix_binding_text = "\n".join(
        row["old_binding"] + "\n" + row["notes"] for row in rows
    )
    ignored_bindings = {"Ping", "ServiceName"}
    missing_bindings = sorted(
        name for name in binding_names - ignored_bindings if name not in matrix_binding_text
    )
    if missing_bindings:
        errors.append("bindings missing from matrix old_binding/notes: " + ", ".join(missing_bindings))

    tools = mcp_tool_names()
    matrix_tool_text = "\n".join(row["old_mcp_tool"] + "\n" + row["notes"] for row in rows)
    missing_tools = sorted(name for name in tools if name not in matrix_tool_text)
    if missing_tools:
        errors.append("MCP tools missing from matrix old_mcp_tool/notes: " + ", ".join(missing_tools))

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1

    print(f"migration matrix OK: {len(rows)} rows, {len(binding_names)} bindings, {len(tools)} MCP tools")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
