#!/usr/bin/env bash
set -euo pipefail
# Test that x --version prints the version from VERSION file
expected=$(<"$(dirname \$0)/../VERSION")
output=$(./x --version)
if [[ "$output" != "$expected" ]]; then
  echo "Expected: $expected"
  echo "Got: $output"
  exit 1
fi
exit 0
