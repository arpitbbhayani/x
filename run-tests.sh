#!/usr/bin/env bash
set -euo pipefail
# Run all test scripts in tests/
for t in tests/*.sh; do
  echo "Running $t"
  bash "$t"
  echo "✅ $t passed"
done
