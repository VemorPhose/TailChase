#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./scripts/release-audit.sh v0.1.28 [--skip-tests]

Runs local release consistency checks. This script does not publish releases,
create GitHub resources, or mutate external state.
USAGE
}

if [[ $# -lt 1 ]]; then
  usage
  exit 2
fi

expected_tag="$1"
shift

skip_tests=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-tests)
      skip_tests=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
  shift
done

if [[ ! "$expected_tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "expected version must look like v0.1.28" >&2
  exit 2
fi

expected_version="${expected_tag#v}"

check_file_mentions() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$file"; then
    echo "$file does not mention $needle" >&2
    exit 1
  fi
}

cli_version="$(sed -nE 's/^const version = "([^"]+)"/\1/p' internal/cli/root.go)"
if [[ "$cli_version" != "$expected_version" ]]; then
  echo "CLI version $cli_version does not match expected $expected_version" >&2
  exit 1
fi

check_file_mentions CHANGELOG.md "$expected_version"
check_file_mentions README.md "$expected_version"
check_file_mentions docs/distribution.md "$expected_version"
check_file_mentions docs/release-readiness.md "$expected_version"

if [[ "$skip_tests" == false ]]; then
  go test ./...
else
  echo "Skipping go test ./... because --skip-tests was provided."
fi

if command -v gh >/dev/null 2>&1; then
  if gh auth status >/dev/null 2>&1; then
    if gh release view "$expected_tag" --repo VemorPhose/TailChase >/dev/null 2>&1; then
      echo "GitHub Release $expected_tag exists."
    else
      echo "GitHub Release $expected_tag was not found or could not be read."
    fi
  else
    echo "gh is installed but not authenticated; skipping GitHub Release check."
  fi
else
  echo "gh is not installed; skipping optional GitHub Release check."
fi

echo "Release audit passed for $expected_tag."
