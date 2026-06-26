#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
gate="${script_dir}/check-go-coverage.sh"
tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/mocktrue-go-coverage.XXXXXX")"
trap 'rm -R "${tmpdir}"' EXIT

below_profile="${tmpdir}/below-threshold.out"
at_threshold_profile="${tmpdir}/at-threshold.out"
duplicate_profile="${tmpdir}/duplicate-blocks.out"
below_log="${tmpdir}/below-threshold.log"
at_threshold_log="${tmpdir}/at-threshold.log"
duplicate_log="${tmpdir}/duplicate-blocks.log"

python3 - "${below_profile}" "${at_threshold_profile}" "${duplicate_profile}" <<'PY'
from pathlib import Path
import sys

below_profile = Path(sys.argv[1])
at_threshold_profile = Path(sys.argv[2])
duplicate_profile = Path(sys.argv[3])

# 2249 covered statements out of 2500 is exactly 89.96%. The old gate parsed
# the rounded `go tool cover -func` total and could treat this as 90.0%.
below_profile.write_text(
    "mode: set\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:1.1,2.1 2249 1\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:2.1,3.1 251 0\n",
    encoding="utf-8",
)

# 2250 covered statements out of 2500 is exactly 90.0% and must pass.
at_threshold_profile.write_text(
    "mode: set\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:1.1,2.1 2250 1\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:2.1,3.1 250 0\n",
    encoding="utf-8",
)

# `go test -coverpkg=...` can emit duplicate blocks. A block covered by any
# package's tests should count covered once, not uncovered/covered duplicates.
duplicate_profile.write_text(
    "mode: set\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:1.1,2.1 1 0\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:1.1,2.1 1 1\n"
    "github.com/littepointR/mocktrue/internal/core/app/app.go:2.1,3.1 1 0\n",
    encoding="utf-8",
)
PY

if GO_COVERAGE_THRESHOLD=90 "${gate}" --enforce-profile "${below_profile}" >"${below_log}" 2>&1; then
  printf 'expected exact 89.96%% coverage to fail threshold 90, but it passed\n' >&2
  printf '%s\n' "$(<"${below_log}")" >&2
  exit 1
fi

below_output="$(<"${below_log}")"
case "${below_output}" in
  *"Go coverage 89.96% is below threshold 90%"*) ;;
  *)
    printf 'unexpected below-threshold output:\n%s\n' "${below_output}" >&2
    exit 1
    ;;
esac

GO_COVERAGE_THRESHOLD=90 "${gate}" --enforce-profile "${at_threshold_profile}" >"${at_threshold_log}" 2>&1

at_threshold_output="$(<"${at_threshold_log}")"
case "${at_threshold_output}" in
  *"Go coverage 90% meets threshold 90%"*) ;;
  *)
    printf 'unexpected at-threshold output:\n%s\n' "${at_threshold_output}" >&2
    exit 1
    ;;
esac

GO_COVERAGE_THRESHOLD=50 "${gate}" --enforce-profile "${duplicate_profile}" >"${duplicate_log}" 2>&1

duplicate_output="$(<"${duplicate_log}")"
case "${duplicate_output}" in
  *"Go coverage 50% meets threshold 50%"*) ;;
  *)
    printf 'unexpected duplicate-block output:\n%s\n' "${duplicate_output}" >&2
    exit 1
    ;;
esac

printf 'Go coverage gate regression checks passed\n'
