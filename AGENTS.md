# Repository Guidelines

## Project Structure & Module Organization
- `cmd/app` holds the API entrypoint (Gin router setup); `cmd/migrate` is the CLI for database migrations (golang-migrate).
- `src/app` contains feature code (`auth`, `models`, `views`); `src/utils` houses shared helpers; `src/config` loads `config.yml` and test config.
- `src/sql` keeps SQL per domain (for queries and migrations). Keep new SQL in the matching feature folder.
- `tests` is an integration-style Ginkgo suite that boots the app against a test database defined in `test.config.yml`.
- `docker-compose.yaml` provides local services; `openapi.yaml` tracks the HTTP contract—update it when endpoints change.

## Build, Test, and Development Commands
- Start dependencies: `docker-compose up -d`.
- Run migrations: `go run cmd/migrate/main.go up`; create one: `go run cmd/migrate/main.go new <name>`.
- Launch the API locally (reads `config.yml`): `go run cmd/app/main.go`.
- Test suite (from repo root): `go test -v ./tests -args -c test.config.yml`; narrow with standard Ginkgo flags (e.g., `-ginkgo.focus`).
- Quick sanity across packages: `go test ./...`.

## Coding Style & Naming Conventions
- Default to idiomatic Go: tabs, `gofmt` clean, small focused functions, lower_snake_case for SQL files, and lower case package names.
- Keep handlers thin and push business logic into `src/app` subpackages; reuse helpers in `src/utils` instead of duplicating logic.
- Prefer descriptive request/response structs; align JSON tags with existing API field names and update `openapi.yaml` when they change.

## Testing Guidelines
- Tests use Ginkgo/Gomega and automatically apply migrations; they drop the test DB afterward. Point `-c` to an isolated DB URL to avoid clobbering local data.
- Add new specs in `tests/<feature>_test.go` with clear `Describe/Context/It` names. Mirror API responses using `gin.H` for expectations.
- Keep data setup minimal; prefer fixtures via SQL helpers in `src/sql` and reuse existing request builders where possible.

## Commit & Pull Request Guidelines
- Commit messages follow short, imperative descriptions (e.g., `Add comprehensive test suite`, `Improve models`).
- PRs should summarize intent, list commands/tests run, call out migration or config changes, and link issues/tasks. Include any contract updates (`openapi.yaml`) and note new environment variables.
- If adding endpoints, mention auth impacts and provide example requests/responses or screenshots from a client if relevant.
