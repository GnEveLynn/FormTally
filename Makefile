DATABASE_URL ?= postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable
TEST_DATABASE_URL ?= $(DATABASE_URL)
ALLOWED_ORIGINS ?= http://127.0.0.1:5173
APP_ENV ?= development
SMS_DRIVER ?= test

.PHONY: dev-api dev-h5 db-up backend-up backend-down migrate migrate-down test-go test-db test-web test-e2e test-miniprogram build-miniprogram scan-miniprogram ai-eval build-release scan-artifacts container-build container-check verify verify-rc

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

backend-up:
	docker compose up -d --build --wait api

backend-down:
	docker compose down

migrate:
	cd server && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	cd server && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

test-go:
	cd server && go test ./...

test-db:
	cd server && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -p 1 ./internal/postgres ./internal/auth ./internal/profile ./internal/goals ./internal/idempotency ./internal/analysis ./internal/meals ./internal/days ./internal/account ./internal/httpapi -count=1

test-web:
	npm --prefix apps/web run test:unit -- --run

test-e2e:
	npm --prefix apps/web run test:e2e

test-miniprogram:
	npm run test:miniprogram

build-miniprogram:
	npm run build:miniprogram

scan-miniprogram: build-miniprogram
	@test -z "$$(find apps/miniprogram/miniprogram -type f \( -name '.env*' -o -name '*.pem' -o -name '*.key' -o -name '*.jpg' -o -name '*.jpeg' -o -name '*.png' -o -name '*.webp' \) -print)"
	@if grep -RIlE 'WECHAT_APP_SECRET|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|Bearer[[:space:]]+[A-Za-z0-9_-]{20,}|1[3-9][0-9]{9}' apps/miniprogram/miniprogram --exclude='*.spec.ts'; then exit 1; fi

ai-eval:
	cd server && go test ./internal/aieval ./cmd/ai-eval -count=1
	cd server && go run ./cmd/ai-eval -manifest ../testdata/ai-quality/manifest.example.json -allow-blocked

build-release:
	npm --prefix apps/web run build
	mkdir -p build
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o ../build/formtally-api ./cmd/api

scan-artifacts: build-release
	@test -z "$$(find build apps/web/dist -type f \( -name '.env*' -o -name '*.pem' -o -name '*.key' -o -name '*.jpg' -o -name '*.jpeg' -o -name '*.png' -o -name '*.webp' \) -print)"
	@if grep -RIlE 'sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|session-cookie-secret|base64-image-secret' build apps/web/dist; then exit 1; fi

container-build:
	docker build -f Containerfile -t formtally:v1-rc .

container-check: container-build
	@uid="$$(docker run --rm --entrypoint /usr/bin/id formtally:v1-rc -u)" && \
		test "$$uid" != 0 && echo "container uid=$$uid"
	docker run --rm --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m \
		-e DATABASE_URL="postgres://formtally:formtally@host.docker.internal:5432/formtally?sslmode=disable" \
		-e ALLOWED_ORIGINS="http://127.0.0.1:5173" \
		-e APP_ENV=development -e SMS_DRIVER=test \
		-e STORAGE_DRIVER=filesystem -e STORAGE_PATH=/tmp/formtally-images \
		-e IMAGE_URL_SECRET=release-check-only \
		formtally:v1-rc /app/formtally-api -check-config
	@cid="$$(docker run --rm -d --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m \
		--add-host host.docker.internal:host-gateway \
		-e DATABASE_URL="postgres://formtally:formtally@host.docker.internal:5432/formtally?sslmode=disable" \
		-e ALLOWED_ORIGINS="http://127.0.0.1:5173" \
		-e APP_ENV=development -e SMS_DRIVER=test \
		-e STORAGE_DRIVER=filesystem -e STORAGE_PATH=/tmp/formtally-images \
		-e IMAGE_URL_SECRET=release-check-only \
		formtally:v1-rc)"; \
	trap 'docker stop "$$cid" >/dev/null 2>&1 || true' EXIT; \
	for attempt in $$(seq 1 20); do \
		status="$$(docker inspect --format '{{.State.Health.Status}}' "$$cid")"; \
		if [ "$$status" = healthy ]; then echo "container health=$$status"; exit 0; fi; \
		if [ "$$status" = unhealthy ]; then docker logs "$$cid"; exit 1; fi; \
		sleep 1; \
	done; \
	docker logs "$$cid"; exit 1

verify: test-go test-db test-web test-e2e test-miniprogram build-miniprogram scan-miniprogram
	npm --prefix apps/web run build

verify-rc: verify ai-eval scan-artifacts container-check
