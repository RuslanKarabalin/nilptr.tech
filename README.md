# nilptr.tech

Personal blog: public posts and pages, anonymous moderated comments,
file downloads and an admin panel at `/admin`.

- product spec (Russian): [docs/SPEC.md](docs/SPEC.md)
- API contract, routing and configuration: [docs/API.md](docs/API.md)
- database schema: [docs/db.sql](docs/db.sql)

## Architecture

| Component | Tech                              | Port | Kubernetes          |
| --------- | --------------------------------- | ---- | ------------------- |
| frontend  | SvelteKit SSR, `adapter-node`     | 3000 | Deployment          |
| backend   | Go REST API, binary `/app/nilptr` | 8080 | Deployment          |
| database  | PostgreSQL 17                     | 5432 | StatefulSet + PVC   |
| files     | Garage (S3 compatible)            | 3900 | StatefulSet + PVCs  |
| ingress   | Traefik (bundled with k3s)        | 443  | Ingress, Middleware |
| TLS       | cert-manager + Let's Encrypt      |      | ClusterIssuer       |

Request flow:

- Traefik routes `/api` and `/files` to the backend, everything else to
  the frontend
- the SvelteKit server renders HTML and calls the backend at
  `BACKEND_URL` (the in-cluster Service)
- PostgreSQL and Garage are never exposed outside the namespace, files
  are always served through the backend
- uploads (up to 1 GiB) stream through SvelteKit, so the frontend has
  `BODY_SIZE_LIMIT=1100M` and Traefik a 30 minute read timeout
- the real client IP reaches the backend through `X-Forwarded-For`
  (Traefik Service with `externalTrafficPolicy: Local`)

Repository layout:

| Path                        | Content                                       |
| --------------------------- | --------------------------------------------- |
| `backend/`                  | Go module, `backend/Dockerfile`               |
| `frontend/`                 | SvelteKit app, `frontend/Dockerfile`          |
| `docker-compose.yaml`       | full local stack                              |
| `deploy/*.env.example`      | local development environment                 |
| `deploy/compose/`           | Caddy proxy and Garage config for compose     |
| `deploy/k8s/bootstrap/`     | one-time cluster objects, applied by hand     |
| `deploy/k8s/secrets/`       | Secret examples only, never real values       |
| `deploy/k8s/base/`          | ConfigMaps, NetworkPolicies, data, backup     |
| `deploy/k8s/migrate/`       | migration Job, recreated on every deploy      |
| `deploy/k8s/app/`           | backend, frontend, Ingress                    |
| `deploy/scripts/`           | deploy, init, kubeconfig, secrets, smoke test |
| `.github/workflows/ci.yaml` | CI/CD                                         |

## Local development

### Full stack with docker compose

Requirements: Docker with Compose v2.24 or newer.

```sh
docker compose up --build
```

Then open <http://localhost:8000> (admin: <http://localhost:8000/admin>,
login `admin`, password `admin`).

Start order, enforced with `depends_on` conditions:

1. `postgres` and `garage` start and become healthy
2. `garage-init` applies the Garage layout, creates the `nilptr` bucket,
   imports the access key from `deploy/backend.env.example` and exits
3. `backend-migrate` runs `/app/nilptr migrate` and exits
4. `backend` starts, `backend-ready` waits for its `/readyz` and exits
5. `frontend` starts and becomes healthy on `/healthz`
6. `proxy` (Caddy) listens on port 8000

The backend image is distroless (no shell, no curl), so it cannot have a
healthcheck of its own. The small `backend-ready` curl container polls
`/readyz` instead, and the frontend and proxy wait for it to finish.

Ports on localhost:

| Port | Service                                             |
| ---- | --------------------------------------------------- |
| 8000 | proxy, use this one, it routes like production      |
| 3000 | frontend directly                                   |
| 8080 | backend directly                                    |
| 5432 | PostgreSQL (`nilptr` / `nilptr`)                    |
| 3900 | Garage S3 API                                       |
| 3903 | Garage admin API (token `dev-admin-token`)          |

In production Traefik sends `/api` and `/files` to the backend. Locally
the `proxy` service does the same, so the browser sees one origin, just
like on `nilptr.tech`. The frontend `ORIGIN` is
`http://localhost:8000`, so form actions only work through the proxy.

Configuration comes from `deploy/backend.env.example` and
`deploy/frontend.env.example` (safe dev values, `COOKIE_SECURE=false`).
To change something locally without touching git, put overrides into
`deploy/.env.backend` or `deploy/.env.frontend`, both are optional and
ignored by git.

Smoke test against the running stack:

```sh
docker compose up --build -d
deploy/scripts/smoke.sh
```

It checks the health endpoints, admin login, creating a published post,
the post in `GET /api/posts` and in the frontend HTML, file upload and
download (including `Range`), the comment limit (second comment gets
429) and `POST /api/views`. The comment limit is per IP and day, so
reset the data before running it again on the same day.

Reset everything (containers and all data):

```sh
docker compose down -v
```

### Running frontend or backend natively

Start only the dependencies:

```sh
docker compose up -d postgres garage garage-init
```

Backend (from `backend/`), with the compose hostnames replaced by
`localhost`:

```sh
set -a
. ../deploy/backend.env.example
set +a
export DATABASE_URL='postgres://nilptr:nilptr@localhost:5432/nilptr?sslmode=disable'
export S3_ENDPOINT=localhost:3900
go run ./cmd/nilptr migrate
go run ./cmd/nilptr serve
```

Frontend (from `frontend/`):

```sh
npm ci
BACKEND_URL=http://localhost:8080 npm run dev
```

The Vite dev server proxies `/api` and `/files` to `BACKEND_URL`
(default `http://localhost:8080`), so <http://localhost:5173> already
routes like production.

To keep the production-like routing on port 8000 with native processes,
point the proxy at the host (Vite dev server runs on 5173):

```sh
BACKEND_UPSTREAM=host.docker.internal:8080 \
FRONTEND_UPSTREAM=host.docker.internal:5173 \
docker compose up -d --no-deps proxy
```

Checks that CI runs:

```sh
# backend/
gofmt -l . && go vet ./... && golangci-lint run && go test ./...
# frontend/
npm ci && npm run lint && npm run check && npm run test && npm run build
```

## CI/CD

Workflow: [.github/workflows/ci.yaml](.github/workflows/ci.yaml).

On pull request:

- backend: gofmt, `go vet`, golangci-lint (`backend/.golangci.yml`),
  `go test -race ./...`
- frontend: `npm ci`, `lint`, `check`, `test`, `build`
- `docker build` of both images without pushing

On push to `main`:

1. the same lint and test jobs
2. build and push `ghcr.io/ruslankarabalin/nilptr-backend` and
   `ghcr.io/ruslankarabalin/nilptr-frontend`, tag = short commit SHA
   (7 chars), with GitHub Actions build cache
3. deploy job (one at a time, `concurrency: deploy-production`) runs
   [deploy/scripts/deploy.sh](deploy/scripts/deploy.sh):
   1. applies `deploy/k8s/base/` (ConfigMaps, NetworkPolicies,
      PostgreSQL, Garage, backup CronJob) and the `garage-init` script
      ConfigMap, waits for PostgreSQL
   2. deletes the previous `backend-migrate` Job (a Job is immutable),
      applies the new one and waits until it is Complete or Failed; on
      failure or timeout it prints the logs and fails the workflow
   3. waits for Garage to become Ready, applies `deploy/k8s/app/`
      (backend, frontend, Ingress) and waits for `kubectl rollout status`
      of backend and frontend

Every manifest goes through `envsubst '${IMAGE_TAG}' | kubectl apply -f -`.
Only `IMAGE_TAG` is substituted, other `$VARS` inside container scripts
stay as they are. The data layer is applied before the migration Job,
because on the very first deploy PostgreSQL does not exist yet.

`deploy/k8s/bootstrap/` and `deploy/k8s/secrets/` are never applied by
CI. The same script works by hand:

```sh
IMAGE_TAG=abc1234 deploy/scripts/deploy.sh
```

## First-time cluster setup

Done once by the cluster owner with cluster-admin access to k3s.

### 1. k3s API reachable from GitHub Actions

The deploy job talks to the k3s API on port 6443. Open the port and make
sure the public address is in the API certificate, for example in
`/etc/rancher/k3s/config.yaml`:

```yaml
tls-san:
  - 203.0.113.10
```

Restart k3s after changing it. Only the `deployer` token is exposed, and
it is limited to the `nilptr` namespace.

### 2. Traefik settings

Keep the client IP and allow long uploads:

```sh
kubectl apply -f deploy/k8s/bootstrap/traefik-helmchartconfig.yaml
kubectl -n kube-system get svc traefik \
  -o jsonpath='{.spec.externalTrafficPolicy}'
```

The second command must print `Local` after k3s redeploys Traefik.

### 3. cert-manager

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
kubectl -n cert-manager rollout status deployment/cert-manager-webhook
```

Edit the email in `deploy/k8s/bootstrap/cluster-issuer.yaml`, then:

```sh
kubectl apply -f deploy/k8s/bootstrap/cluster-issuer.yaml
```

DNS: `nilptr.tech` must point to the server, and port 80 must be open
for the HTTP-01 challenge.

### 4. Namespace and deployer RBAC

```sh
kubectl apply -f deploy/k8s/bootstrap/namespace.yaml
kubectl apply -f deploy/k8s/bootstrap/rbac-deployer.yaml
```

### 5. Secrets

Never commit real values. `deploy/k8s/secrets/*.example.yaml` only show
the shape and the exact `kubectl create secret` commands.

Application secrets with random values (`postgres`, `garage`,
`backend`):

```sh
ADMIN_PASSWORD='a long password' deploy/scripts/create-secrets.sh
```

Image pull secret (GitHub token, classic, with only `read:packages`):

```sh
kubectl -n nilptr create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username=ruslankarabalin \
  --docker-password='<token>'
```

Backup target (see [Backups](#backups)):

```sh
kubectl -n nilptr create secret generic backup-target \
  --from-literal=BACKUP_S3_ENDPOINT=https://s3.example.com \
  --from-literal=BACKUP_S3_REGION=us-east-1 \
  --from-literal=BACKUP_S3_BUCKET=nilptr-backup \
  --from-literal=BACKUP_S3_ACCESS_KEY='<key>' \
  --from-literal=BACKUP_S3_SECRET_KEY='<secret>'
```

Notes:

- the PostgreSQL password is used only when the volume is empty; to
  change it later run `ALTER ROLE` and update both `postgres` and
  `DATABASE_URL` in `backend`
- the Garage access key (`S3_ACCESS_KEY`, `GK` + 24 hex chars) is
  imported once; Garage never accepts the same key id again, so rotate it
  by setting a new key id and secret and redeploying

### 6. Kubeconfig for GitHub Actions

```sh
deploy/scripts/make-kubeconfig.sh https://203.0.113.10:6443 > deployer.kubeconfig
KUBECONFIG=deployer.kubeconfig kubectl -n nilptr get pods
gh secret set KUBECONFIG < deployer.kubeconfig
rm deployer.kubeconfig
```

The token comes from the `deployer-token` Secret and does not expire. To
rotate it, delete that Secret, apply `rbac-deployer.yaml` again and
repeat this step.

### 7. First deploy

Push to `main` (or run the workflow by hand). The first deploy creates
PostgreSQL and Garage, then the `garage-init` init container of the
migration Job applies the Garage layout and creates the bucket and key,
then migrations run and the apps start. Check:

```sh
kubectl -n nilptr get pods,pvc,ingress,certificate
kubectl -n nilptr logs job/backend-migrate -c garage-init
curl -I https://nilptr.tech/
```

## Garage

Garage serves 503 on `/health` until it has a cluster layout, so a fresh
install needs a one-time init. It is automated by
[deploy/scripts/garage-init.sh](deploy/scripts/garage-init.sh), which
uses the Garage admin API (v2) with curl and is idempotent:

1. if the layout version is 0, assign the single node to zone `dc1` and
   apply layout version 1
2. create the bucket `S3_BUCKET` if missing
3. import the key `S3_ACCESS_KEY` / `S3_SECRET_KEY` if missing
4. allow the key read, write and owner on the bucket

It runs as `garage-init` in docker compose and as the first init
container of the `backend-migrate` Job on every deploy. The init script
reaches the admin API through the `garage-headless` Service, which
publishes the pod before it is Ready.

Manual inspection, the image has no shell but the binary can be run:

```sh
kubectl -n nilptr exec garage-0 -- /garage status
kubectl -n nilptr exec garage-0 -- /garage bucket info nilptr
```

## Backups

All data lives on one disk, so the `backup` CronJob
([deploy/k8s/base/backup-cronjob.yaml](deploy/k8s/base/backup-cronjob.yaml))
copies it to external S3-compatible storage every night at 03:30 UTC:

- `pg_dump --format=custom` to
  `<bucket>/nilptr/postgres/nilptr-<UTC time>.dump`
- every object of the Garage bucket to `<bucket>/nilptr/files/` with
  `rclone copy` (objects deleted in Garage are kept in the backup)
- dumps older than `BACKUP_RETENTION` (30 days) are removed

The target is configured in the `backup-target` Secret, prefix and
retention in the `backup-config` ConfigMap. Use a dedicated bucket and a
key limited to it, ideally with object versioning on the provider side.

Run a backup now and watch it:

```sh
kubectl -n nilptr create job --from=cronjob/backup backup-manual-$(date +%s)
kubectl -n nilptr get jobs -l app.kubernetes.io/name=backup
kubectl -n nilptr logs -l app.kubernetes.io/name=backup --all-containers --prefix
```

### Restore

Example with rclone on your machine configured with a remote `backup:`
for the target and a remote `garage:` for the cluster Garage
(`provider = Other`, `region = garage`, `force_path_style = true`,
`endpoint = http://localhost:3900`, keys from the `backend` Secret).

1. Stop writers:

   ```sh
   kubectl -n nilptr scale deployment/backend deployment/frontend --replicas=0
   kubectl -n nilptr patch cronjob backup -p '{"spec":{"suspend":true}}'
   ```

2. Restore the database from the chosen dump:

   ```sh
   rclone lsf backup:nilptr-backup/nilptr/postgres/
   DUMP=nilptr-20261006T033000Z.dump
   rclone cat "backup:nilptr-backup/nilptr/postgres/$DUMP" \
     | kubectl -n nilptr exec -i postgres-0 -- sh -c \
       'pg_restore --clean --if-exists --no-owner \
          -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
   ```

3. Restore the files. On a new cluster deploy once first, so Garage has
   its layout, bucket and key. Then:

   ```sh
   kubectl -n nilptr port-forward svc/garage 3900:3900 &
   rclone copy -P backup:nilptr-backup/nilptr/files/ garage:nilptr/
   kill %1
   ```

4. Start again:

   ```sh
   kubectl -n nilptr scale deployment/backend deployment/frontend --replicas=1
   kubectl -n nilptr patch cronjob backup -p '{"spec":{"suspend":false}}'
   ```

Test a restore from time to time, for example into the local compose
stack (`localhost:5432` and `localhost:3900`).

## Security notes

- all containers run as non-root with all capabilities dropped and,
  where possible, a read-only root filesystem
- NetworkPolicies (enforced by the k3s built-in controller):
  - PostgreSQL accepts only the backend, the migration Job and the
    backup
  - Garage S3 accepts only the backend and the backup, the Garage admin
    API only pods of the namespace
  - the frontend accepts only Traefik and can reach only the backend
  - the backend accepts only Traefik and the frontend
- the CI `deployer` ServiceAccount can manage workloads in `nilptr` only
  and cannot read Secrets
