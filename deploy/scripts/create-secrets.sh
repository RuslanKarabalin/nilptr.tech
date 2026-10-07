#!/usr/bin/env bash
# Creates the application Secrets in namespace nilptr with random values:
# postgres, garage and backend. Existing Secrets are never overwritten
# (changing the database password or the Garage key after the first start
# needs extra steps, see README.md).
#
#   ADMIN_PASSWORD='a long password' deploy/scripts/create-secrets.sh
#
# Optional: ADMIN_LOGIN (default admin).
# Not created here: ghcr-pull and backup-target, they need external
# credentials (see deploy/k8s/secrets/*.example.yaml).
set -euo pipefail

NS=nilptr
: "${ADMIN_PASSWORD:?set ADMIN_PASSWORD}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"

exists() { kubectl -n "$NS" get secret "$1" >/dev/null 2>&1; }
hex() { openssl rand -hex "$1"; }

if exists postgres; then
  echo "secret postgres exists, keeping it"
else
  kubectl -n "$NS" create secret generic postgres \
    --from-literal=POSTGRES_USER=nilptr \
    --from-literal=POSTGRES_DB=nilptr \
    --from-literal=POSTGRES_PASSWORD="$(hex 24)"
fi

if exists garage; then
  echo "secret garage exists, keeping it"
else
  kubectl -n "$NS" create secret generic garage \
    --from-literal=GARAGE_RPC_SECRET="$(hex 32)" \
    --from-literal=GARAGE_ADMIN_TOKEN="$(hex 32)" \
    --from-literal=GARAGE_METRICS_TOKEN="$(hex 32)"
fi

if exists backend; then
  echo "secret backend exists, keeping it"
else
  pg_user=$(kubectl -n "$NS" get secret postgres -o jsonpath='{.data.POSTGRES_USER}' | base64 --decode)
  pg_pass=$(kubectl -n "$NS" get secret postgres -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 --decode)
  pg_db=$(kubectl -n "$NS" get secret postgres -o jsonpath='{.data.POSTGRES_DB}' | base64 --decode)
  kubectl -n "$NS" create secret generic backend \
    --from-literal=DATABASE_URL="postgres://${pg_user}:${pg_pass}@postgres:5432/${pg_db}?sslmode=disable" \
    --from-literal=JWT_SECRET="$(hex 32)" \
    --from-literal=IP_HASH_SECRET="$(hex 32)" \
    --from-literal=ADMIN_LOGIN="$ADMIN_LOGIN" \
    --from-literal=ADMIN_PASSWORD="$ADMIN_PASSWORD" \
    --from-literal=S3_ACCESS_KEY="GK$(hex 12)" \
    --from-literal=S3_SECRET_KEY="$(hex 32)"
fi

echo "done. Still needed: ghcr-pull and backup-target secrets."
