#!/usr/bin/env bash
# End-to-end smoke test against a running local stack:
#
#   docker compose up --build -d && deploy/scripts/smoke.sh
#
# Checks health endpoints, admin login, post creation, the post in the
# public API and in the frontend HTML, file upload and download, the one
# comment per IP per day limit and POST /api/views.
#
# The comment limit is per IP and UTC day and survives restarts, so a
# second run on the same day fails at the comment step unless the data is
# reset with: docker compose down -v
#
# Environment (defaults match docker-compose.yaml):
#   BASE_URL      proxy, like production    http://localhost:8000
#   BACKEND_URL   backend direct            http://localhost:8080
#   FRONTEND_URL  frontend direct           http://localhost:3000
#   ADMIN_LOGIN / ADMIN_PASSWORD            admin / admin
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
JAR="$WORK/cookies.txt"
BODY="$WORK/body"
SLUG="smoke-$(date +%s)"
FAILED=0

pass() { printf 'PASS  %s\n' "$*"; }
fail() {
  printf 'FAIL  %s\n' "$*"
  if [ -s "$BODY" ]; then
    printf '      response: %s\n' "$(head -c 400 "$BODY")"
  fi
  FAILED=1
}

# req METHOD URL [curl args...] - prints the HTTP status, body in $BODY.
req() {
  local method="$1" url="$2"
  shift 2
  : >"$BODY"
  curl -sS -o "$BODY" -w '%{http_code}' -X "$method" \
    -b "$JAR" -c "$JAR" "$@" "$url" || echo 000
}

expect() {
  local want="$1" got="$2" what="$3"
  if [ "$got" = "$want" ]; then pass "$what ($got)"; else fail "$what: want $want, got $got"; fi
}

json_field() {
  # First string value of a top-level-ish JSON field, enough for smoke.
  sed -n "s/.*\"$1\" *: *\"\\([^\"]*\\)\".*/\\1/p" "$BODY" | head -n 1
}

echo "== health"
expect 200 "$(req GET "$BACKEND_URL/healthz")" "backend /healthz"
expect 200 "$(req GET "$BACKEND_URL/readyz")" "backend /readyz"
expect 200 "$(req GET "$FRONTEND_URL/healthz")" "frontend /healthz"
expect 200 "$(req GET "$FRONTEND_URL/readyz")" "frontend /readyz"
expect 200 "$(req GET "$BASE_URL/")" "proxy / (frontend)"

echo "== admin login"
status=$(req POST "$BASE_URL/api/admin/login" -H 'Content-Type: application/json' \
  -d "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}")
expect 204 "$status" "POST /api/admin/login"
if grep -q access_token "$JAR" && grep -q refresh_token "$JAR"; then
  pass "access_token and refresh_token cookies set"
else
  fail "login cookies missing"
fi
expect 200 "$(req GET "$BASE_URL/api/admin/me")" "GET /api/admin/me"
expect 401 "$(curl -sS -o /dev/null -w '%{http_code}' "$BASE_URL/api/admin/me")" \
  "GET /api/admin/me without cookies"

echo "== posts"
status=$(req POST "$BASE_URL/api/admin/posts" -H 'Content-Type: application/json' \
  -d "{\"slug\":\"$SLUG\",\"title\":\"Smoke $SLUG\",\"body\":\"Hello from **smoke** test.\",\"status\":\"published\"}")
expect 201 "$status" "POST /api/admin/posts ($SLUG)"
status=$(req GET "$BASE_URL/api/posts?limit=100")
if [ "$status" = 200 ] && grep -q "\"$SLUG\"" "$BODY"; then
  pass "GET /api/posts lists $SLUG"
else
  fail "GET /api/posts does not list $SLUG (status $status)"
fi
expect 200 "$(req GET "$BASE_URL/api/posts/$SLUG")" "GET /api/posts/$SLUG"
status=$(req GET "$BASE_URL/posts/$SLUG")
if [ "$status" = 200 ] && grep -q "Smoke $SLUG" "$BODY"; then
  pass "frontend /posts/$SLUG renders the title"
else
  fail "frontend /posts/$SLUG (status $status) does not contain the title"
fi

echo "== files"
upload="$WORK/smoke file.txt"
printf 'smoke test file %s\n' "$SLUG" >"$upload"
status=$(req POST "$BASE_URL/api/admin/files" -F "file=@$upload;type=text/plain")
expect 201 "$status" "POST /api/admin/files"
file_id=$(json_field id)
file_url=$(json_field url)
if [ -n "$file_id" ] && [ -n "$file_url" ]; then
  pass "upload returned id $file_id and url $file_url"
  status=$(req GET "$BASE_URL$file_url")
  if [ "$status" = 200 ] && cmp -s "$BODY" "$upload"; then
    pass "GET $file_url returns the same content"
  else
    fail "GET $file_url (status $status) content differs"
  fi
  status=$(req GET "$BASE_URL/files/$file_id")
  expect 200 "$status" "GET /files/$file_id"
  status=$(curl -sS -o /dev/null -w '%{http_code}' -H 'Range: bytes=0-4' "$BASE_URL/files/$file_id")
  expect 206 "$status" "GET /files/$file_id with Range"
else
  fail "upload response has no id or url"
fi

echo "== comments"
status=$(req POST "$BASE_URL/api/posts/$SLUG/comments" -H 'Content-Type: application/json' \
  -d '{"author":"smoke","body":"first comment","website":""}')
expect 201 "$status" "first comment"
if [ "$status" = 429 ]; then
  echo "      hint: this IP already commented today, reset with docker compose down -v"
fi
status=$(req POST "$BASE_URL/api/posts/$SLUG/comments" -H 'Content-Type: application/json' \
  -d '{"author":"smoke","body":"second comment","website":""}')
expect 429 "$status" "second comment is rate limited"

echo "== views"
status=$(req POST "$BASE_URL/api/views" -H 'Content-Type: application/json' \
  -d "{\"path\":\"/posts/$SLUG\"}")
expect 204 "$status" "POST /api/views"

echo
if [ "$FAILED" -ne 0 ]; then
  echo "SMOKE TEST FAILED"
  exit 1
fi
echo "SMOKE TEST PASSED"
