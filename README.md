# FileForce

FileForce is a small, extensible Go-based skeleton that demonstrates a minimal "Google Drive"-style backend: file upload, listing, download, and delete, with a simple authentication portal. This repository is intended as a starting point you can extend for production use.

**Location:** the runnable server is in `src/beta`.

**Important:** This skeleton is for development and learning. It includes a basic in-memory session store and SHA-256 salted password hashing. For production you should use secure password hashing (bcrypt/argon2), persistent session storage, TLS, and other hardening steps listed below.

**Table of contents**

- **Quick Start**
- **Endpoints**
- **Authentication**
- **File types and extension support**
- **Examples (curl)**
- **Configuration & development**
- **Security notes & recommended improvements**
- **Next steps / ideas**

## Quick Start

Run the server locally:

```bash
cd src/beta
go run .
```

By default the server listens on `:8080` and stores files in a `data/` directory created next to where you run the server.

Default test credential (development only):

- username: `admin`
- password: `password`

The server creates that user automatically on first run (password is stored as a salted SHA-256 hash).

## Endpoints

- `GET /login` - human login form
- `POST /api/login` - API login form (expects `username` and `password` form fields). Sets an HTTP-only `session` cookie on success.
- `GET /logout` - clears session and redirects to `/login`
- `POST /upload` - authenticated endpoint. Multipart form field `file` to upload (protected by session cookie).
- `GET /files` - lists all uploaded files (JSON array)
- `GET /files/{name}` - downloads the file with the given name
- `DELETE /files/{name}` - deletes the file (requires authentication)

## Authentication (how it works)

- Login is served at `/login` (HTML). The form posts to `/api/login`.
- On successful login the server creates a random token mapped to the username in an in-memory session map and sets a `session` cookie. The cookie is `HttpOnly` and expires in 24 hours.
- The `RequireAuth` middleware protects the `/upload` endpoint and the delete operation. Public endpoints like `/files` (listing and downloads) remain accessible without auth by default.

Notes: this version uses `bcrypt` for password hashing and supports persistent sessions in Redis. The server will try to initialize Redis on startup using the `REDIS_ADDR` environment variable (default `localhost:6379`). If Redis is unavailable the server logs a warning and operates without persistent sessions.

## File types and extension support

The server enforces a simple allowlist for file extensions. Supported extensions include common types and the niche `.tscn` extension required for some game/editor files. The whitelist is in `src/beta/main.go` as `allowedExts`.

To allow additional extensions, edit `allowedExts` in `src/beta/main.go` and restart the server.

## Redis (sessions)

FileForce can persist sessions in Redis for durability and multi-instance deployments. To run Redis locally with Docker:

```bash
docker run -d --name fileforce-redis -p 6379:6379 redis:7
```

Start the Go server with a `REDIS_ADDR` environment variable if Redis is not on the default host/port:

```bash
cd src/beta
REDIS_ADDR=localhost:6379 go run .
```

When Redis is connected the server stores session tokens there; if Redis is not reachable the server logs a warning and continues (sessions are not persisted across restarts).

## UI / Static files

The repository provides minimal UI pages for authentication:

- `GET /login` — login form (served from `src/beta/templates/login.html`)
- `GET /register` — registration form (served from `src/beta/templates/register.html`)

Static assets (CSS/JS) live in `src/beta/static` and are served from `/static/`.

## Examples (curl)

Login and store cookies:

```bash
curl -c cookies.txt -d "username=admin&password=password" -X POST http://localhost:8080/api/login
```

Upload (authenticated):

```bash
curl -b cookies.txt -F "file=@path/to/your/file.tscn" http://localhost:8080/upload
```

List files (public):

```bash
curl http://localhost:8080/files
```

Download a file:

```bash
curl -O "http://localhost:8080/files/yourfile.tscn"
```

Delete a file (authenticated):

```bash
curl -b cookies.txt -X DELETE "http://localhost:8080/files/yourfile.tscn"
```

## Configuration & development

- Go version: `go 1.20` is specified in `src/beta/go.mod`.
- Build/run from `src/beta`:

```bash
cd src/beta
go run .
```

- Files are stored under the `data/` directory created at runtime.

## Security notes & recommended improvements

The current implementation is intentionally minimal. Before using in any public or production environment, consider implementing the following:

- Use a slow password hashing algorithm (`bcrypt`, `argon2`) instead of raw SHA-256.
- Store users and sessions in a database (Postgres/SQLite) or a session store (Redis) rather than in-memory maps.
- Serve the app over HTTPS and set cookie `Secure` flag.
- Add CSRF protection for browser forms.
- Add upload size limits and per-user quotas to prevent disk exhaustion.
- Add logging and audit trails for uploads, downloads and deletes (who/when).
- Add input validation and content scanning where appropriate.

## Next steps / ideas

- Persist user accounts and add registration flow.
- Add per-user storage namespaces and file metadata (owner, upload time, content-type).
- Implement resumable/chunked uploads for large files.
- Implement role-based access control, sharing links, and file versioning.
- Swap salted SHA-256 for `bcrypt` or `argon2` (strongly recommended).

## New: registration, per-user directories, and Rust component

- This version includes user registration (`POST /api/register`) and a password-change endpoint (`POST /api/change-password`).
- Files are stored in per-user directories under `data/users/<username>/` when uploaded by authenticated users. The listing endpoint `GET /files` returns the current user's files by default. Use `GET /files?all=1` to list across users (admin use).
- A small Rust component skeleton is included at `rust_component/` as an example worker (indexer/thumbnailer/async worker) you can extend. Build it with `cargo build` inside that directory.

## How this compares to other file-sharing tools

- Dropbox / Google Drive: full-featured clients, sync engines, real-time collaboration, and mature sharing controls. FileForce is a minimal backend skeleton focused on file storage APIs — it lacks sync clients, offline conflict resolution, and collaborative editing.
- Nextcloud: offers self-hosted file storage with rich plugins, WebDAV, and user management. FileForce is a lightweight starting point; Nextcloud provides federation, apps, and production-ready components out of the box.
- S3-backed solutions: scale and durability by design. FileForce stores files on disk by default; integrating S3 (or other object stores) is a recommended next step for durability and scalability.

FileForce is intentionally small so you can prototype storage workflows, auth flows, and optional Rust workers for CPU-bound tasks (thumbnailing, scanning, indexing) while keeping the core simple.

## Running locally with Docker Compose

You can run the Go server and Redis together using the provided `docker-compose.yml` at the repository root.

```bash
docker compose up --build
```

This mounts `./src/beta` into the Go container so code changes take effect immediately (it runs `go run .` inside the container). Redis will be available at `redis:6379` and the Go server will be reachable at `http://localhost:8080`.

## Rust and C helpers (download clients)

Two small client utilities are included to demonstrate saving files to the computer:

- Rust downloader: `workers` is a Rust Cargo project. Build and run the downloader:

```bash
cd workers
cargo run -- download http://localhost:8080/files/yourfile.tscn /tmp/yourfile.tscn
```

- C downloader: a tiny libcurl-based program is in `workers/c_client`. Build with a working libcurl installation:

```bash
cd workers/c_client
make
./download http://localhost:8080/files/yourfile.tscn /tmp/yourfile.tscn
```

These clients are minimal examples showing how to programmatically fetch files from FileForce and save them locally. Extend them to support authentication (cookie handling or token headers) as needed.

## Contributing

Pull requests and suggestions welcome. Open an issue to discuss larger changes.

## License

See the repository `LICENSE` file.

