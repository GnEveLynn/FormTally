DATABASE_URL ?= postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable
TEST_DATABASE_URL ?= $(DATABASE_URL)
ALLOWED_ORIGINS ?= http://127.0.0.1:5173
APP_ENV ?= development
SMS_DRIVER ?= test

.PHONY: dev-api dev-h5 db-up migrate migrate-down test-go test-db test-web test-e2e verify

dev-api:
	cd server && DATABASE_URL="$(DATABASE_URL)" ALLOWED_ORIGINS="$(ALLOWED_ORIGINS)" APP_ENV="$(APP_ENV)" SMS_DRIVER="$(SMS_DRIVER)" go run ./cmd/api

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
	cd server && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./internal/postgres ./internal/auth ./internal/profile ./internal/goals ./internal/idempotency ./internal/analysis ./internal/meals ./internal/days -count=1

test-web:
	npm --prefix apps/web run test:unit -- --run

test-e2e:
	npm --prefix apps/web run test:e2e

verify: test-go test-db test-web test-e2e
	npm --prefix apps/web run build
