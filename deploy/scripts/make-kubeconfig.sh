#!/usr/bin/env bash
# Prints a kubeconfig for the "deployer" ServiceAccount (namespace nilptr)
# to stdout. Run it with cluster-admin access after applying
# deploy/k8s/bootstrap/rbac-deployer.yaml:
#
#   deploy/scripts/make-kubeconfig.sh https://203.0.113.10:6443 > deployer.kubeconfig
#   gh secret set KUBECONFIG < deployer.kubeconfig
#   rm deployer.kubeconfig
#
# The server address must be reachable from GitHub Actions and must be in
# the k3s API certificate (k3s --tls-san <address> for a public IP or DNS
# name).
set -euo pipefail

SERVER="${1:?usage: $0 https://<public address>:6443}"
NS=nilptr
SA=deployer
SECRET=deployer-token

token=""
for _ in $(seq 1 30); do
  token=$(kubectl -n "$NS" get secret "$SECRET" -o jsonpath='{.data.token}' 2>/dev/null || true)
  [ -n "$token" ] && break
  sleep 1
done
[ -n "$token" ] || { echo "secret $NS/$SECRET has no token yet" >&2; exit 1; }
token=$(printf '%s' "$token" | base64 --decode)
ca=$(kubectl -n "$NS" get secret "$SECRET" -o jsonpath='{.data.ca\.crt}')

cat <<EOF
apiVersion: v1
kind: Config
clusters:
  - name: nilptr
    cluster:
      server: ${SERVER}
      certificate-authority-data: ${ca}
users:
  - name: ${SA}
    user:
      token: ${token}
contexts:
  - name: ${SA}@nilptr
    context:
      cluster: nilptr
      user: ${SA}
      namespace: ${NS}
current-context: ${SA}@nilptr
EOF
