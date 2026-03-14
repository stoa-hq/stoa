#!/bin/sh
# Tests for docker-plugins.sh
# Run: sh scripts/docker-plugins_test.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILURES=0

fail() {
  echo "FAIL: $1"
  FAILURES=$((FAILURES + 1))
}

pass() {
  echo "PASS: $1"
}

# --- Test: resolve_plugin function ---

# Source resolve_plugin and is_local_path by extracting them
eval "$(sed -n '/^resolve_plugin/,/^}/p' "$SCRIPT_DIR/docker-plugins.sh")"
eval "$(sed -n '/^is_local_path/,/^}/p' "$SCRIPT_DIR/docker-plugins.sh")"

# Test known short names
result=$(resolve_plugin "n8n")
if [ "$result" = "github.com/stoa-hq/stoa-plugins/n8n" ]; then
  pass "resolve_plugin n8n"
else
  fail "resolve_plugin n8n: got '$result'"
fi

result=$(resolve_plugin "stripe")
if [ "$result" = "github.com/stoa-hq/stoa-plugins/stripe" ]; then
  pass "resolve_plugin stripe"
else
  fail "resolve_plugin stripe: got '$result'"
fi

# Test unknown name passes through
result=$(resolve_plugin "github.com/example/custom-plugin")
if [ "$result" = "github.com/example/custom-plugin" ]; then
  pass "resolve_plugin passthrough"
else
  fail "resolve_plugin passthrough: got '$result'"
fi

# --- Test: is_local_path ---

if is_local_path "./plugins/myplugin"; then
  pass "is_local_path ./relative"
else
  fail "is_local_path ./relative"
fi

if is_local_path "../external-plugin"; then
  pass "is_local_path ../parent"
else
  fail "is_local_path ../parent"
fi

if is_local_path "/absolute/path"; then
  pass "is_local_path /absolute"
else
  fail "is_local_path /absolute"
fi

if is_local_path "github.com/example/plugin"; then
  fail "is_local_path remote (should be false)"
else
  pass "is_local_path remote (false)"
fi

if is_local_path "n8n"; then
  fail "is_local_path shortname (should be false)"
else
  pass "is_local_path shortname (false)"
fi

# --- Test: empty input exits cleanly ---

output=$(sh "$SCRIPT_DIR/docker-plugins.sh" "" 2>&1)
if echo "$output" | grep -q "No plugins requested"; then
  pass "empty input skips"
else
  fail "empty input: got '$output'"
fi

# --- Test: no arguments exits cleanly ---

output=$(sh "$SCRIPT_DIR/docker-plugins.sh" 2>&1)
if echo "$output" | grep -q "No plugins requested"; then
  pass "no arguments skips"
else
  fail "no arguments: got '$output'"
fi

# --- Test: local path not found exits with error ---

output=$(sh "$SCRIPT_DIR/docker-plugins.sh" "./nonexistent-plugin" 2>&1) && rc=0 || rc=$?
if [ "$rc" -ne 0 ] && echo "$output" | grep -q "local plugin directory not found"; then
  pass "missing local dir errors"
else
  fail "missing local dir: rc=$rc, got '$output'"
fi

# --- Summary ---
echo ""
if [ "$FAILURES" -gt 0 ]; then
  echo "$FAILURES test(s) failed."
  exit 1
else
  echo "All tests passed."
fi
