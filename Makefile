DATABASE_URL ?= postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable
TEST_DATABASE_URL ?= $(DATABASE_URL)
ALLOWED_ORIGINS ?= http://127.0.0.1:5173

.PHONY: dev-api dev-h5 db-up migrate migrate-down test-go test-db test-web test-e2e verify

dev-api:
	cd server && DATABASE_URL="$(DATABASE_URL)" ALLOWED_ORIGINS="$(ALLOWED_ORIGINS)" go run ./cmd/api

dev-h5:
	npm --workspace @formtally/web run dev -- --host 127.0.0.1

db-up:
	@if docker inspect formtally-postgres >/dev/null 2>&1; then \
		test "$$(docker inspect --format '{{.State.Health.Status}}' formtally-postgres)" = healthy; \
	else \
		docker compose up -d --wait postgres; \
	fi

migrate:
	cd server && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	cd server && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

test-go:
	cd server && go test ./...

test-db:
	cd server && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./internal/postgres -count=1

test-web:
	npm --prefix apps/web run test:unit -- --run

test-e2e:
	@echo "H5 journey tests will be available in Task 4." >&2
	@exit 2

# Expand this gate when browser journey suites are implemented.
verify: test-go test-db test-web
	npm --prefix apps/web run build
