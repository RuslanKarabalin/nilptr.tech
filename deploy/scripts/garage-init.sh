#!/bin/sh
# Idempotent Garage bootstrap through the admin API (v2):
#   1. assign a role to the single node and apply the cluster layout
#   2. create the bucket
#   3. import the access key from S3_ACCESS_KEY / S3_SECRET_KEY
#   4. allow the key to read, write and own the bucket
#
# Needs only POSIX sh, curl and sed, so it runs in curlimages/curl both in
# docker compose and in the backend-migrate Job. Safe to run on every
# deploy.
#
# Environment:
#   GARAGE_ADMIN_URL    admin API base, e.g. http://garage:3903
#   GARAGE_ADMIN_TOKEN  admin bearer token
#   S3_BUCKET           bucket name (global alias)
#   S3_ACCESS_KEY       key id, "GK" + 24 hex chars
#   S3_SECRET_KEY       secret, 64 hex chars
#   GARAGE_ZONE         layout zone (default dc1)
#   GARAGE_CAPACITY     node capacity in bytes (default 50 GB)
#   GARAGE_WAIT_SECONDS how long to wait for the admin API (default 300)
set -eu

: "${GARAGE_ADMIN_URL:?}" "${GARAGE_ADMIN_TOKEN:?}" "${S3_BUCKET:?}"
: "${S3_ACCESS_KEY:?}" "${S3_SECRET_KEY:?}"
ZONE="${GARAGE_ZONE:-dc1}"
CAPACITY="${GARAGE_CAPACITY:-50000000000}"
WAIT="${GARAGE_WAIT_SECONDS:-300}"
API="${GARAGE_ADMIN_URL%/}/v2"
BODY="${TMPDIR:-/tmp}/garage-init.$$"
trap 'rm -f "$BODY"' EXIT

log() { echo "garage-init: $*"; }

# call METHOD PATH [JSON] - prints the HTTP status, body goes to $BODY.
call() {
  if [ $# -ge 3 ]; then
    curl -sS -o "$BODY" -w '%{http_code}' -X "$1" \
      -H "Authorization: Bearer $GARAGE_ADMIN_TOKEN" \
      -H 'Content-Type: application/json' -d "$3" "$API/$2"
  else
    curl -sS -o "$BODY" -w '%{http_code}' -X "$1" \
      -H "Authorization: Bearer $GARAGE_ADMIN_TOKEN" "$API/$2"
  fi
}

fail() {
  log "ERROR: $*"
  cat "$BODY" 2>/dev/null || true
  echo
  exit 1
}

# First 64 hex char "id" field of the JSON in $BODY.
first_id() {
  sed -n 's/.*"id": *"\([0-9a-f]\{64\}\)".*/\1/p' "$BODY" | head -n 1
}

log "waiting for $API"
i=0
until status=$(call GET GetClusterStatus 2>/dev/null) && [ "$status" = 200 ]; do
  i=$((i + 2))
  [ "$i" -lt "$WAIT" ] || fail "admin API not reachable after ${WAIT}s"
  sleep 2
done

# 1. layout
version=$(sed -n 's/.*"layoutVersion": *\([0-9][0-9]*\).*/\1/p' "$BODY" | head -n 1)
node=$(first_id)
[ -n "$node" ] || fail "cannot find node id"
if [ "${version:-0}" -eq 0 ]; then
  log "assigning role to node $node (zone $ZONE, capacity $CAPACITY)"
  roles="{\"roles\":[{\"id\":\"$node\",\"zone\":\"$ZONE\",\"capacity\":$CAPACITY,\"tags\":[]}]}"
  [ "$(call POST UpdateClusterLayout "$roles")" = 200 ] || fail "UpdateClusterLayout"
  [ "$(call POST ApplyClusterLayout '{"version":1}')" = 200 ] || fail "ApplyClusterLayout"
  log "layout version 1 applied"
else
  log "layout already applied (version $version)"
fi

# 2. bucket
status=$(call GET "GetBucketInfo?globalAlias=$S3_BUCKET")
case "$status" in
  200) log "bucket $S3_BUCKET exists" ;;
  404)
    [ "$(call POST CreateBucket "{\"globalAlias\":\"$S3_BUCKET\"}")" = 200 ] ||
      fail "CreateBucket"
    log "bucket $S3_BUCKET created"
    ;;
  *) fail "GetBucketInfo returned $status" ;;
esac
bucket_id=$(first_id)
[ -n "$bucket_id" ] || fail "cannot find bucket id"

# 3. key
status=$(call GET "GetKeyInfo?id=$S3_ACCESS_KEY")
case "$status" in
  200) log "key $S3_ACCESS_KEY exists" ;;
  404)
    key="{\"accessKeyId\":\"$S3_ACCESS_KEY\",\"secretAccessKey\":\"$S3_SECRET_KEY\",\"name\":\"$S3_BUCKET-app\"}"
    [ "$(call POST ImportKey "$key")" = 200 ] || fail "ImportKey"
    log "key $S3_ACCESS_KEY imported"
    ;;
  *) fail "GetKeyInfo returned $status" ;;
esac

# 4. permissions (setting the same permissions again is a no-op)
allow="{\"bucketId\":\"$bucket_id\",\"accessKeyId\":\"$S3_ACCESS_KEY\",\"permissions\":{\"read\":true,\"write\":true,\"owner\":true}}"
[ "$(call POST AllowBucketKey "$allow")" = 200 ] || fail "AllowBucketKey"
log "key has read, write and owner on $S3_BUCKET"
log "done"
