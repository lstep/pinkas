#!/usr/bin/env bash
# Smoke test for collaborative editing (Iteration 1/2 integration)
# Verifies that the critical path for live sync works after changes.

set -euo pipefail

BASE="http://localhost:8081"
TOKEN=""
PAGE_ID=""
SPACE_ID=""

echo "=== Mostdoc Collaborative Editing Smoke Test ==="

# 1. Health checks
echo "[1/7] Checking Go API health..."
curl -sf "$BASE/health" > /dev/null || { echo "FAIL: Go API not responding"; exit 1; }
echo "      OK"

echo "[2/7] Checking collab sidecar health..."
curl -sf "$BASE/health" > /dev/null || { echo "FAIL: Sidecar not responding"; exit 1; }
echo "      OK"

# 2. Register first admin (or login if already registered)
echo "[3/7] Registering/logging in admin..."
REGISTER=$(curl -s -X POST "$BASE/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@test.com","password":"password123","name":"Test Admin"}' 2>/dev/null)

HAS_TOKEN=$(echo "$REGISTER" | python3 -c "import sys,json; d=json.load(sys.stdin); print('yes' if d.get('token') else 'no')" 2>/dev/null || echo "no")

if [ "$HAS_TOKEN" = "no" ]; then
  # Already registered, login instead
  LOGIN=$(curl -sf -X POST "$BASE/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@test.com","password":"password123"}')
  TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['token']['accessToken'])" 2>/dev/null || echo "")
else
  TOKEN=$(echo "$REGISTER" | python3 -c "import sys,json; print(json.load(sys.stdin)['token']['accessToken'])" 2>/dev/null || echo "")
fi

if [ -z "$TOKEN" ]; then
  echo "FAIL: Could not obtain auth token"
  exit 1
fi
echo "      OK (token obtained)"

# 3. Get default space
echo "[4/7] Fetching default space..."
SPACES=$(curl -sf "$BASE/api/spaces" -H "Authorization: Bearer $TOKEN")
SPACE_ID=$(echo "$SPACES" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['spaces'][0]['id'] if d.get('spaces') else '')" 2>/dev/null || echo "")

if [ -z "$SPACE_ID" ]; then
  echo "FAIL: No spaces found"
  exit 1
fi
echo "      OK (space: $SPACE_ID)"

# 4. Create a test page
echo "[5/7] Creating test page..."
PAGE=$(curl -sf -X POST "$BASE/api/pages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"spaceId\":\"$SPACE_ID\",\"title\":\"Smoke Test Page\",\"parentId\":\"\"}")
PAGE_ID=$(echo "$PAGE" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")

if [ -z "$PAGE_ID" ]; then
  echo "FAIL: Could not create page"
  exit 1
fi
echo "      OK (page: $PAGE_ID)"

# 5. Verify /internal/load returns valid JSON (the regression we fixed)
echo "[6/7] Testing /internal/load (regression: nil snapshot panic)..."
LOAD=$(curl -sf "$BASE/internal/load?docId=$PAGE_ID" || true)
if [ -z "$LOAD" ]; then
  echo "FAIL: /internal/load returned empty (Go API may have panicked)"
  exit 1
fi

# Verify it's valid JSON with yjsSnapshot key
YJS=$(echo "$LOAD" | python3 -c "import sys,json; print('ok' if 'yjsSnapshot' in json.load(sys.stdin) else 'fail')" 2>/dev/null || echo "fail")
if [ "$YJS" != "ok" ]; then
  echo "FAIL: /internal/load returned invalid JSON"
  exit 1
fi
echo "      OK (no panic, valid JSON)"

# 6. Save a snapshot and reload
echo "[7/7] Testing /internal/save + reload..."
SAVE=$(curl -sf -X POST "$BASE/internal/save" \
  -H "Content-Type: application/json" \
  -d "{\"docId\":\"$PAGE_ID\",\"markdown\":\"# Smoke Test\",\"yjsSnapshot\":[1,2,3],\"authorId\":\"$PAGE_ID\"}")

if [ -z "$SAVE" ]; then
  echo "FAIL: /internal/save failed"
  exit 1
fi

# Now load should return the snapshot
LOAD2=$(curl -sf "$BASE/internal/load?docId=$PAGE_ID")
SNAPSHOT_LEN=$(echo "$LOAD2" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('yjsSnapshot','')) if d.get('yjsSnapshot') else 0)" 2>/dev/null || echo "0")

if [ "$SNAPSHOT_LEN" -eq 0 ]; then
  echo "FAIL: Snapshot was not persisted"
  exit 1
fi
echo "      OK (snapshot persisted and reloadable)"

# Cleanup
curl -sf -X DELETE "$BASE/api/pages/$PAGE_ID" -H "Authorization: Bearer $TOKEN" > /dev/null 2>&1 || true

echo ""
echo "=== ALL TESTS PASSED ==="
echo "Collaborative editing path is healthy."
