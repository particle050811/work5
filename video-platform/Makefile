APP_NAME := fanone-video
SERVER_LOG := /tmp/$(APP_NAME)-e2e.log
E2E_BIN := /tmp/$(APP_NAME)-e2e

.PHONY: help init tidy build run test e2e e2e-full docker-build

help:
	@echo "可用命令："
	@echo "  make init         初始化本地开发环境（生成 .env、创建存储目录、下载依赖）"
	@echo "  make tidy         整理并下载 Go 依赖"
	@echo "  make build        构建本地二进制"
	@echo "  make run          启动服务"
	@echo "  make test         运行 video-platform 单元测试"
	@echo "  make e2e          运行 test/ 目录端到端测试"
	@echo "  make e2e-full     自动启动服务并执行完整 e2e"
	@echo "  make docker-build 构建 Docker 镜像"

init:
	@bash script/init.sh

tidy:
	@go mod tidy
	@go mod download

build:
	@go build -o $(APP_NAME) .

run:
	@go run .

test:
	@go test ./...

e2e:
	@cd ../test && go run .

e2e-full:
	@set -e; \
	if lsof -ti tcp:8888 >/dev/null 2>&1; then \
		echo "端口 8888 已被占用，请先停止现有服务再执行 make e2e-full"; \
		exit 1; \
	fi; \
	status=1; \
	ready=0; \
	go build -o $(E2E_BIN) .; \
	$(E2E_BIN) > $(SERVER_LOG) 2>&1 & \
	server_pid=$$!; \
	trap 'kill $$server_pid >/dev/null 2>&1 || true; rm -f $(E2E_BIN)' EXIT INT TERM; \
	for i in $$(seq 1 20); do \
		if curl --noproxy localhost -fsS http://localhost:8888/ping >/dev/null 2>&1; then \
			ready=1; \
			if cd ../test && go run .; then \
				status=0; \
			else \
				status=$$?; \
			fi; \
			break; \
		fi; \
		sleep 1; \
	done; \
	if [ $$ready -eq 0 ]; then \
		echo "服务启动超时，日志如下:"; \
		tail -n 50 $(SERVER_LOG); \
		status=1; \
	fi; \
	kill $$server_pid >/dev/null 2>&1 || true; \
	rm -f $(E2E_BIN); \
	exit $$status

docker-build:
	@docker build -t $(APP_NAME):latest .
