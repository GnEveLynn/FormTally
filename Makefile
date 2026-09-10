.PHONY: dev-api dev-h5 test-go test-db test-web test-e2e verify

dev-api:
	cd server && go run ./cmd/api

dev-h5:
	npm --workspace @formtally/web run dev -- --host 127.0.0.1

test-go:
	cd server && go test ./...

test-db:
	@echo "PostgreSQL integration tests will be available in Task 2." >&2
	@exit 2

test-web:
	npm --prefix apps/web run test:unit -- --run

test-e2e:
	@echo "H5 journey tests will be available in Task 4." >&2
	@exit 2

# Expand this gate when database and browser journey suites are implemented.
verify: test-go test-web
	npm --prefix apps/web run build
