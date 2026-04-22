.PHONY: help build run run-local kill test test-verbose lint fix docker-up docker-down db-create db-migrate db-seed db-setup db-reset db-state clean

ENV_FILE ?= .env.local

-include .env $(ENV_FILE)
export

MIGRATE_CMD    = go run -tags 'mysql' -mod=mod github.com/golang-migrate/migrate/v4/cmd/migrate@latest
ADMIN_DB_URL   = mysql://root:$(MYSQL_ROOT_PASSWORD)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/$(MYSQL_DATABASE)
DOCKER_COMPOSE = docker compose
DOCKER_MYSQL_ADMIN = $(DOCKER_COMPOSE) exec -T mysql mysql -u root -p$(MYSQL_ROOT_PASSWORD)
DOCKER_MYSQL_APP   = $(DOCKER_COMPOSE) exec -T mysql mysql -u $(MYSQL_USER) -p$(MYSQL_PASSWORD) $(MYSQL_DATABASE)
APP_USER_HOST  ?= %

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build        バイナリをビルド"
	@echo "  run          ローカル API を起動"
	@echo "  run-local    ローカル API を起動"
	@echo "  test         テストを実行"
	@echo "  test-verbose テストを詳細出力で実行"
	@echo "  lint         lint を実行"
	@echo "  fix          lint を実行し自動修正"
	@echo "  docker-up    ローカル開発用 MySQL コンテナを起動"
	@echo "  docker-down  ローカル開発用 MySQL コンテナを停止"
	@echo "  db-create    DB を作成"
	@echo "  db-migrate   マイグレーションを実行"
	@echo "  db-seed      シードデータを投入"
	@echo "  db-setup     マイグレーションとシードをまとめて実行"
	@echo "  db-reset     DB をリセットして再構築"
	@echo "  db-state     マイグレーションの適用状況を表示"
	@echo "  kill         API サーバーを停止"
	@echo "  clean        ビルド成果物を削除"
	@echo "  help         このヘルプを表示"

build:
	go build -o bin/api ./app/main.go

clean:
	rm -rf bin/

run: run-local

run-local:
	go run ./app/main.go

kill:
	@set -a; \
	if [ -f "$(ENV_FILE)" ]; then . "$(ENV_FILE)"; fi; \
	set +a; \
	port=$${PORT:-8080}; \
	pids=$$(lsof -tiTCP:$$port -sTCP:LISTEN); \
	if [ -n "$$pids" ]; then \
		for pid in $$pids; do \
			pgid=$$(ps -o pgid= -p $$pid | tr -d ' '); \
			if [ -n "$$pgid" ]; then kill -TERM -$$pgid 2>/dev/null || true; fi; \
			kill -TERM $$pid 2>/dev/null || true; \
		done; \
		sleep 1; \
		stubborn=$$(lsof -tiTCP:$$port -sTCP:LISTEN); \
		if [ -n "$$stubborn" ]; then kill -KILL $$stubborn 2>/dev/null || true; fi; \
	fi

test:
	go test ./...

test-verbose:
	go test -v ./...

lint:
	golangci-lint run ./...

fix:
	golangci-lint run --fix ./...

docker-up:
	$(DOCKER_COMPOSE) up -d --wait

docker-down:
	$(DOCKER_COMPOSE) down

db-setup: db-create-user db-migrate db-seed

db-create:
	$(DOCKER_MYSQL_ADMIN) \
		-e "CREATE DATABASE IF NOT EXISTS $(MYSQL_DATABASE);"

db-create-user:
	$(DOCKER_MYSQL_ADMIN) \
		-e "CREATE USER IF NOT EXISTS '$(MYSQL_USER)'@'$(APP_USER_HOST)' IDENTIFIED BY '$(MYSQL_PASSWORD)'; GRANT ALL PRIVILEGES ON $(MYSQL_DATABASE).* TO '$(MYSQL_USER)'@'$(APP_USER_HOST)'; FLUSH PRIVILEGES;"

db-migrate:
	$(MIGRATE_CMD) -path db/migrate -database "$(ADMIN_DB_URL)" up

db-seed:
	for f in db/seed/*.sql; do \
		$(DOCKER_MYSQL_ADMIN) < $$f; \
	done

db-reset:
	@if [ "$(APP_ENV)" != "development" ]; then \
		echo "Error: db-reset is only allowed in development environment (APP_ENV=development)."; \
		exit 1; \
	fi
	$(DOCKER_MYSQL_ADMIN) \
		-e "DROP DATABASE IF EXISTS $(MYSQL_DATABASE); CREATE DATABASE $(MYSQL_DATABASE);"
	$(MAKE) db-migrate db-seed

db-state:
	@echo "Status | Version            | Migration"
	@echo "-------|--------------------|--------------------------"
	@current=$$($(DOCKER_MYSQL_APP) \
		-s --skip-column-names -e "SELECT version FROM schema_migrations;" 2>/dev/null); \
	for f in db/migrate/*.up.sql; do \
		version=$$(basename $$f | cut -d_ -f1); \
		name=$$(basename $$f .up.sql | cut -d_ -f2-); \
		if [ -n "$$current" ] && [ "$$version" -le "$$current" ]; then \
			printf "up     | %s | %s\n" "$$version" "$$name"; \
		else \
			printf "down   | %s | %s\n" "$$version" "$$name"; \
		fi; \
	done
