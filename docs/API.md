# API

Contract between the SvelteKit frontend and the Go backend.
It complements [SPEC.md](SPEC.md). If something here differs from
SPEC.md, this file wins.

## Routing

Traefik Ingress for `nilptr.tech`:

- `/api` and `/files` go to the backend Service (port 8080)
- everything else goes to the frontend Service (port 3000)

The SvelteKit server talks to the backend at `BACKEND_URL`
(in the cluster `http://backend.nilptr.svc.cluster.local:8080`).

The browser calls the backend directly only for:

- `POST /api/views`
- `POST /api/posts/{slug}/comments`
- `GET /files/...`

All admin actions go through SvelteKit (`/admin/...` pages, form
actions and server endpoints), which calls `/api/admin/...` on the
backend server-side.

## General rules

- JSON in and out, `Content-Type: application/json`
- errors: HTTP status plus body `{"error": "human readable message"}`
- timestamps: RFC 3339 strings in UTC
- ids: `uuid` strings, except `nav_items`, `comments` and `page_views`
  which use integer ids
- list endpoints return `{"items": [...], "total": N}` when paginated
  (`limit` default 20, max 100, `offset` default 0)

## Client IP

- backend reads the client IP from `X-Forwarded-For`, walking from the
  right and skipping addresses from `TRUSTED_PROXIES` (CIDR list)
- if the direct peer is not trusted, the peer address is used
- SvelteKit, when calling the backend on behalf of a browser
  (login, refresh), sends `X-Forwarded-For` with the client address it
  received and the original `User-Agent`

## Health

- `GET /healthz` - 200 if the process is alive
- `GET /readyz` - 200 if PostgreSQL and S3 are reachable, else 503

The frontend exposes the same two paths (readyz may just return 200).

## Public endpoints

### Posts

`GET /api/posts?limit=&offset=` - published posts, newest first by
`published_at`.

```json
{
  "items": [
    {"slug": "hello", "title": "Hello", "published_at": "2026-10-06T10:00:00Z"}
  ],
  "total": 1
}
```

`GET /api/posts/{slug}` - post with status `published` or `unlisted`,
otherwise 404.

```json
{
  "slug": "hello",
  "title": "Hello",
  "body": "markdown",
  "status": "published",
  "published_at": "2026-10-06T10:00:00Z",
  "updated_at": "2026-10-06T10:00:00Z"
}
```

### Comments

`GET /api/posts/{slug}/comments` - approved comments, oldest first:
`[{"id": 1, "author": "anon", "body": "text", "created_at": "..."}]`.
`author` may be `null`.

`POST /api/posts/{slug}/comments`

```json
{
  "author": "optional, max 64 chars",
  "body": "required, max 4000 chars",
  "website": ""
}
```

- `website` is the honeypot: if not empty, respond 201 and store nothing
- 404 if the post is not `published` or `unlisted`
- 429 if this IP already submitted a comment today (UTC day)
- 201 `{"status": "pending"}` on success

### Pages

`GET /api/pages/{slug}` - `{"slug", "title", "body", "updated_at"}` or 404.

### Navigation

`GET /api/nav`

```json
{
  "header": [{"label": "CV", "url": "/cv"}],
  "footer": [{"label": "GitHub", "url": "https://github.com/RuslanKarabalin"}]
}
```

Items are sorted by `position`. A `url` starting with `/` is internal.

### Files

`GET /api/files/{id}` - metadata
`{"id", "name", "content_type", "size", "url"}`, where `url` is
`/files/{id}/{name}` (name URL-escaped).

`GET /files/{id}/{name}` and `GET /files/{id}` - file content streamed
from S3.

- supports `Range` and `HEAD` (needed for video)
- `Content-Type` from the stored value
- images, video and audio are served `inline`, everything else as
  `attachment` with the stored file name
- `Cache-Control: public, max-age=31536000, immutable`
- the `{name}` part is ignored for lookup, only `{id}` matters

### Views

`POST /api/views` `{"path": "/posts/hello"}` - 204.

- `path` must start with `/`, max 512 chars, must not start with `/admin`
- simple in-memory rate limit per IP (for example 60 per minute),
  excess requests get 204 and are dropped

## Admin endpoints

Everything under `/api/admin` requires a valid access token, except
`login` and `refresh`. Missing or invalid token gives 401.

### Cookies

| Cookie          | Content              | Path     | Max-Age    |
| --------------- | -------------------- | -------- | ---------- |
| `access_token`  | JWT HS256, 15 min    | `/`      | 900        |
| `refresh_token` | random 32 bytes, b64 | `/admin` | 30 days    |

- both `HttpOnly`, `SameSite=Strict`, `Secure` unless
  `COOKIE_SECURE=false` (local development)
- the refresh cookie path is `/admin` (not `/api/admin/refresh` as in
  the first version of SPEC.md) so that the browser sends it to the
  SvelteKit admin pages, and SvelteKit can refresh server-side
- SvelteKit forwards the cookies to the backend in the `Cookie` header
  and relays `Set-Cookie` headers from the backend to the browser
- JWT claims: `sub` (user id), `sid` (session id = refresh token
  `family_id`), `exp`, `iat`
- the access token is accepted only while its session (family) is not
  revoked, so ending a session takes effect immediately

### Auth

- `POST /api/admin/login` `{"login", "password"}` - 204 and both
  cookies, 401 on bad credentials. Uses `User-Agent` and client IP to
  create a new session (family).
- `POST /api/admin/refresh` - reads `refresh_token` cookie, 204 and
  both cookies rotated. On failure 401 and both cookies cleared.
  - token unknown or expired: 401
  - user-agent differs from the stored one: 401
  - token already revoked by rotation less than 10 seconds ago
    (concurrent refresh from two tabs): 401, the family is NOT revoked
    and the cookies are NOT cleared (no `Set-Cookie`), otherwise this
    response would delete the cookies just set by the winning request
  - token already revoked otherwise: revoke the whole family, 401
- `POST /api/admin/logout` - revokes the current session, clears both
  cookies, 204
- `GET /api/admin/me` - `{"id", "login"}`

### Sessions

- `GET /api/admin/sessions` -
  `[{"id", "user_agent", "ip", "created_at", "last_used_at", "current"}]`,
  one item per active family
- `DELETE /api/admin/sessions/{id}` - revoke the family, 204

### Admin posts

- `GET /api/admin/posts?limit=&offset=&status=` - all posts, newest
  first by `updated_at`: items
  `{"id", "slug", "title", "status", "created_at", "updated_at", "published_at"}`
- `POST /api/admin/posts` `{"slug", "title", "body", "status"}` - 201
  with full post (including `id` and `body`)
- `GET /api/admin/posts/{id}` - full post
- `PUT /api/admin/posts/{id}` - same body as create, returns full post
- `DELETE /api/admin/posts/{id}` - 204

Rules: `slug` matches `^[a-z0-9]+(-[a-z0-9]+)*$`, unique (409 on
conflict). `published_at` is set the first time status becomes
`published` and is kept afterwards.

### Admin pages

Same as posts under `/api/admin/pages`, without `status` and
`published_at`.

### Admin navigation

- `GET /api/admin/nav` - `{"header": [...], "footer": [...]}`,
  items `{"id", "label", "url"}`
- `PUT /api/admin/nav` - same shape without `id`, replaces all items
  in one transaction, position is the index in the array

### Admin files

- `GET /api/admin/files?limit=&offset=` - items
  `{"id", "name", "content_type", "size", "created_at", "url"}`
- `POST /api/admin/files` - `multipart/form-data`, field `file`,
  max size `MAX_UPLOAD_BYTES` (default 1 GiB), 201 with the item
- `DELETE /api/admin/files/{id}` - deletes object and row, 204

### Admin comments

- `GET /api/admin/comments?status=pending&limit=&offset=` - items
  `{"id", "post_slug", "post_title", "author", "body", "status", "created_at"}`
- `POST /api/admin/comments/{id}/approve` - 204
- `POST /api/admin/comments/{id}/reject` - 204
- `DELETE /api/admin/comments/{id}` - 204

### Stats

`GET /api/admin/stats?from=2026-10-01&to=2026-10-06` (dates inclusive,
UTC, default last 30 days) -
`{"items": [{"path": "/", "count": 42}], "total": 42}` sorted by count.

## Markdown

Rendered by SvelteKit on the server (also used for the admin preview):
unified + remark-parse + remark-gfm + remark-math + remark-directive +
remark-rehype + rehype-katex + rehype-sanitize + rehype-stringify.

Directives (leaf directives, `::name{id=...}`):

- `::image{id=UUID alt="text"}` - `<img src="/files/{id}" alt loading="lazy">`
- `::video{id=UUID}` - `<video controls preload="metadata" src="/files/{id}">`
- `::file{id=UUID}` - download card: file name, size, link to the
  file `url`; the renderer fetches metadata via `GET /api/files/{id}`

The sanitizer schema must allow KaTeX output and the elements above.

## Configuration

Backend environment variables:

| Name               | Example                                  | Secret |
| ------------------ | ---------------------------------------- | ------ |
| `HTTP_ADDR`        | `:8080`                                  | no     |
| `DATABASE_URL`     | see below                                | yes    |
| `JWT_SECRET`       | 32+ random bytes                         | yes    |
| `IP_HASH_SECRET`   | 32+ random bytes                         | yes    |
| `ADMIN_LOGIN`      | `admin`                                  | yes    |
| `ADMIN_PASSWORD`   | creates or updates the admin on start    | yes    |
| `S3_ENDPOINT`      | `garage:3900`                            | no     |
| `S3_BUCKET`        | `nilptr`                                 | no     |
| `S3_ACCESS_KEY`    |                                          | yes    |
| `S3_SECRET_KEY`    |                                          | yes    |
| `S3_REGION`        | `garage`                                 | no     |
| `S3_USE_SSL`       | `false`                                  | no     |
| `TRUSTED_PROXIES`  | `10.42.0.0/16,127.0.0.1/32`              | no     |
| `COOKIE_SECURE`    | `true`                                   | no     |
| `MAX_UPLOAD_BYTES` | `1073741824`                             | no     |

S3 storage is [Garage](https://garagehq.deuxfleurs.fr/). The bucket and
the access key are created once by an init step with the `garage` CLI
(`garage bucket create`, `garage key import` or `garage key create`,
`garage bucket allow --read --write`). The backend does not create the
bucket, it only checks on startup and in `/readyz` that the bucket is
reachable. The S3 client must use path-style addressing and the
region from `S3_REGION` (Garage default is `garage`).

`DATABASE_URL` example:
`postgres://nilptr:pw@postgres:5432/nilptr?sslmode=disable`.

The backend binary has subcommands: `serve` (default) and `migrate`
(applies embedded SQL migrations and exits, used by the migration Job).

Frontend environment variables (adapter-node):

| Name             | Example                                          |
| ---------------- | ------------------------------------------------ |
| `PORT`           | `3000`                                           |
| `ORIGIN`         | `https://nilptr.tech`                            |
| `BACKEND_URL`    | `http://backend.nilptr.svc.cluster.local:8080`   |
| `ADDRESS_HEADER` | `X-Forwarded-For`                                |
| `XFF_DEPTH`      | `1`                                              |
