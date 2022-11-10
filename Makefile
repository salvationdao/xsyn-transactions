PACKAGE=xsyn-transactions

# Names and Versions
DOCKER_CONTAINER=$(PACKAGE)-db
MIGRATE_VERSION=v4.12.2

# Paths
BIN = $(CURDIR)/bin
SERVER = $(CURDIR)

# DB Settings
LOCAL_DEV_DB_USER?=$(PACKAGE)-db
LOCAL_DEV_DB_PASS?=dev
LOCAL_DEV_DB_HOST?=localhost
LOCAL_DEV_DB_PORT?=5433
LOCAL_DEV_DB_DATABASE?=$(PACKAGE)-db
DB_CONNECTION_STRING="postgres://$(LOCAL_DEV_DB_USER):$(LOCAL_DEV_DB_PASS)@$(LOCAL_DEV_DB_HOST):$(LOCAL_DEV_DB_PORT)/$(LOCAL_DEV_DB_DATABASE)?sslmode=disable"

# Versions
BUF_VERSION=1.9.0

.PHONY: init
init: tools go-mod-tidy docker-serve-dev

.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy

.PHONY: db-reset
db-reset: db-drop db-migrate db-boiler migrate-from-old

.PHONY: db-drop
db-drop:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path $(CURDIR)/migrations drop -f

.PHONY: db-migrate
db-migrate:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path $(CURDIR)/migrations up

.PHONY: db-migrate-down
db-migrate-down:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path $(CURDIR)/migrations down

.PHONY: db-migrate-down-one
db-migrate-down-one:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path $(CURDIR)/migrations down 1

.PHONY: db-migrate-up-one
db-migrate-up-one:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path $(CURDIR)/migrations up 1

.PHONY: tools
tools:
	@mkdir -p $(BIN)
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	go generate -tags tools ./tools/...


.PHONY: db-boiler
db-boiler:
	$(BIN)/sqlboiler $(BIN)/sqlboiler-psql --wipe --config ${SERVER}/sqlboiler.toml


.PHONY: buf
buf:
	rm -rf gen
	buf generate

.PHONY: migrate-from-old
migrate-from-old:
	go run ./cmd/migrate/main.go

.PHONY: serve
serve:
	${BIN}/air -c ./.air.toml

.PHONY: docker-serve-dev
docker-serve-dev:
	docker compose --profile dev up --detach

.PHONY: docker-stop-dev
docker-stop-dev:
	docker compose --profile dev stop

.PHONY: docker-serve
docker-serve:
	docker compose --profile serve up -d

.PHONY: docker-stop
docker-stop:
	docker compose stop

.PHONY: docker-stop-api
docker-stop-api:
	docker compose --profile api stop

.PHONY: docker-reset
docker-reset: docker-down docker-serve

.PHONY: docker-build
docker-build:
	 docker build -t ninja-syndicate/xsyn-transactions:develop -f ./DockerfileXsynTransactions .
	 docker build -t ninja-syndicate/xsyn-transactions-migrate:develop -f ./DockerfileMigrate .

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-restart-api
docker-restart-api:
	docker compose --profile api restart
