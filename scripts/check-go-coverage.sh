#!/usr/bin/env bash
set -euo pipefail

threshold="${GO_COVERAGE_THRESHOLD:-90}"
profile="${GO_COVERAGE_PROFILE:-coverage.out}"

usage() {
  cat >&2 <<'EOF'
Usage:
  scripts/check-go-coverage.sh
  scripts/check-go-coverage.sh --enforce-profile <coverprofile>

The default mode runs backend/internal Go tests, writes a coverage profile,
prints the human-readable go tool cover summary, and enforces the coverage
threshold using exact statement counts parsed from the cover profile.
EOF
}

coverage_from_profile() {
  python3 - "$1" <<'PY'
from decimal import Decimal, getcontext
from pathlib import Path
import sys

getcontext().prec = 40
profile = Path(sys.argv[1])

try:
    lines = profile.read_text(encoding="utf-8").splitlines()
except OSError as exc:
    raise SystemExit(f"Unable to read Go coverage profile {profile}: {exc}")

blocks = {}
for line_number, line in enumerate(lines, start=1):
    line = line.strip()
    if not line or line.startswith("mode:"):
        continue

    parts = line.split()
    if len(parts) != 3:
        raise SystemExit(
            f"Unable to parse Go coverage profile {profile}:{line_number}: {line!r}"
        )

    location = parts[0]
    try:
        statements = int(parts[1])
        count = int(parts[2])
    except ValueError as exc:
        raise SystemExit(
            f"Invalid Go coverage profile counts at {profile}:{line_number}: {line!r}"
        ) from exc

    if statements < 0 or count < 0:
        raise SystemExit(
            f"Negative Go coverage profile counts at {profile}:{line_number}: {line!r}"
        )

    # `go test -coverpkg=... ./internal/...` can emit the same instrumented block
    # once per tested package. Merge duplicate blocks and mark a block covered if
    # any package's tests covered it, matching the aggregate reported by
    # `go tool cover -func` while keeping the threshold comparison unrounded.
    key = (location, statements)
    blocks[key] = max(blocks.get(key, 0), count)

if not blocks:
    raise SystemExit(f"Go coverage profile {profile} contains no statements")

total_statements = sum(statements for (_location, statements) in blocks)
covered_statements = sum(
    statements for (_location, statements), count in blocks.items() if count > 0
)

coverage = Decimal(covered_statements) * Decimal(100) / Decimal(total_statements)
print(format(coverage, "f"))
PY
}

enforce_threshold() {
  local coverage="$1"
  python3 - "${coverage}" "${threshold}" <<'PY'
from decimal import Decimal, InvalidOperation
import sys

try:
    coverage = Decimal(sys.argv[1])
    threshold = Decimal(sys.argv[2])
except InvalidOperation as exc:
    raise SystemExit(f"Invalid Go coverage threshold input: {exc}") from exc

coverage_text = format(coverage, "f")
threshold_text = format(threshold, "f")
if coverage < threshold:
    raise SystemExit(
        f"Go coverage {coverage_text}% is below threshold {threshold_text}%"
    )

print(f"Go coverage {coverage_text}% meets threshold {threshold_text}%")
PY
}

case "${1:-}" in
  "")
    # Keep this focused on checked-in backend/internal packages. The repository root
    # embeds the frontend dist, so package-wide `go test ./...` coverage can fail in
    # clean source checkouts before the frontend is built.
    go test -coverpkg=./internal/... -coverprofile="${profile}" ./internal/...
    go tool cover -func="${profile}"
    coverage="$(coverage_from_profile "${profile}")"
    enforce_threshold "${coverage}"
    ;;
  --enforce-profile)
    if [ "$#" -ne 2 ]; then
      usage
      exit 2
    fi
    profile="$2"
    coverage="$(coverage_from_profile "${profile}")"
    enforce_threshold "${coverage}"
    ;;
  -h|--help)
    usage
    ;;
  *)
    usage
    exit 2
    ;;
esac
