#!/usr/bin/env bash
# Deploys the manifests in deploy/k8s to the current kubectl context.
# Used by GitHub Actions (.github/workflows/ci.yaml) and usable by hand:
#
#   IMAGE_TAG=abc1234 deploy/scripts/deploy.sh
#
# Order:
#   1. base: ConfigMaps, NetworkPolicies, PostgreSQL, Garage, backup CronJob
#      and the garage-init script ConfigMap; wait for PostgreSQL
#   2. delete the previous backend-migrate Job, apply the new one, wait
#      until it is Complete or Failed (logs are printed in both cases)
#   3. wait for Garage, apply app: backend, frontend, Ingress; wait for
#      both rollouts
#
# Not applied here (manual, cluster owner): deploy/k8s/bootstrap/*
# (Namespace, RBAC, ClusterIssuer, Traefik HelmChartConfig) and
# deploy/k8s/secrets/* (examples only).
set -euo pipefail

: "${IMAGE_TAG:?IMAGE_TAG must be set, e.g. the short commit SHA}"
export IMAGE_TAG
NS=nilptr
MIGRATE_TIMEOUT="${MIGRATE_TIMEOUT:-600}"
ROLLOUT_TIMEOUT="${ROLLOUT_TIMEOUT:-300s}"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
K8S="$ROOT/deploy/k8s"

apply() {
  # Substitute only IMAGE_TAG, so shell variables inside container
  # scripts (for example $BACKUP_PREFIX) stay untouched.
  local f
  for f in "$@"; do
    echo "::group::apply ${f#"$ROOT"/}"
    # shellcheck disable=SC2016 # literal on purpose, envsubst expands it
    envsubst '${IMAGE_TAG}' <"$f" | kubectl apply -f -
    echo "::endgroup::"
  done
}

migrate_logs() {
  kubectl -n "$NS" logs -l job-name=backend-migrate --all-containers \
    --prefix --tail=-1 || true
}

echo "deploying IMAGE_TAG=$IMAGE_TAG"

# 1. data layer and shared config
apply "$K8S"/base/*.yaml
kubectl -n "$NS" create configmap garage-init \
  --from-file=garage-init.sh="$ROOT/deploy/scripts/garage-init.sh" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$NS" rollout status statefulset/postgres --timeout="$ROLLOUT_TIMEOUT"

# 2. migrations (a Job is immutable, so replace it)
kubectl -n "$NS" delete job backend-migrate --ignore-not-found --wait=true
apply "$K8S/migrate/job.yaml"

echo "waiting for job/backend-migrate (up to ${MIGRATE_TIMEOUT}s)"
deadline=$((SECONDS + MIGRATE_TIMEOUT))
result=""
while [ "$SECONDS" -lt "$deadline" ]; do
  complete=$(kubectl -n "$NS" get job backend-migrate \
    -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}')
  failed=$(kubectl -n "$NS" get job backend-migrate \
    -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}')
  if [ "$complete" = "True" ]; then
    result=complete
    break
  fi
  if [ "$failed" = "True" ]; then
    result=failed
    break
  fi
  sleep 5
done

echo "::group::backend-migrate logs"
migrate_logs
echo "::endgroup::"
if [ "$result" != "complete" ]; then
  echo "::error::backend-migrate ${result:-timed out}"
  kubectl -n "$NS" describe job backend-migrate || true
  kubectl -n "$NS" get pods -l job-name=backend-migrate -o wide || true
  exit 1
fi
echo "backend-migrate complete"

# 3. stateless apps and ingress (garage is Ready once its layout exists,
# the backend fails fast without it)
kubectl -n "$NS" rollout status statefulset/garage --timeout="$ROLLOUT_TIMEOUT"
apply "$K8S"/app/*.yaml
kubectl -n "$NS" rollout status deployment/backend --timeout="$ROLLOUT_TIMEOUT"
kubectl -n "$NS" rollout status deployment/frontend --timeout="$ROLLOUT_TIMEOUT"
echo "deploy of $IMAGE_TAG finished"
