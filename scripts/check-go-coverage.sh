#!/usr/bin/env bash
set -euo pipefail

threshold="${GO_COVERAGE_THRESHOLD:-90}"
profile="${GO_COVERAGE_PROFILE:-coverage.out}"

# Keep this focused on checked-in backend/internal packages. The repository root
# embeds the frontend dist, so package-wide `go test ./...` coverage can fail in
# clean source checkouts before the frontend is built.
go test -coverpkg=./internal/... -coverprofile="${profile}" ./internal/...
summary=$(go tool cover -func="${profile}")
printf '%s\n' "${summary}"
coverage=$(printf '%s\n' "${summary}" | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')

if [ -z "${coverage}" ]; then
  echo "Unable to parse Go coverage total" >&2
  exit 1
fi

python3 - <<PY
coverage = float("${coverage}")
threshold = float("${threshold}")
if coverage + 1e-9 < threshold:
    raise SystemExit(f"Go coverage {coverage:.1f}% is below threshold {threshold:.1f}%")
print(f"Go coverage {coverage:.1f}% meets threshold {threshold:.1f}%")
PY
