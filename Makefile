.PHONY: all deps tidy test test-race test-integration test-mutation test-oauth \
	build build-all build-examples run fmt fmt-check vet lint vuln clean install docs \
	example-simple example-webhook example-api example-advanced example-complete \
	verify test-full help

GO ?= go
GOFMT ?= gofmt
PROJECT_NAME := kook-go-sdk
BUILD_DIR := build
EXAMPLES := simple_bot webhook_bot api_usage advanced_bot complete_api_demo

all: test build-all

# 下载 go.mod 中声明的依赖，不修改 go.mod 或 go.sum。
deps:
	$(GO) mod download

# 整理依赖；仅在有意更新依赖时运行。
tidy:
	$(GO) mod tidy

# 默认测试完全离线，不需要 KOOK 凭据。
test:
	$(GO) test -short ./...

test-race:
	$(GO) test -race -short ./...

# 运行真实 KOOK 只读测试，所需环境变量见 DETAILED_GUIDE.md。
test-integration:
	$(GO) test ./...

# 该目标会创建、更新并删除真实资源，应使用隔离测试服务器。
test-mutation:
	KOOK_ENABLE_MUTATION_TESTS=1 $(GO) test ./kook -run '^TestKOOKMutationIntegration$$' -count=1 -v

# OAuth 授权码通常只能使用一次。
test-oauth:
	KOOK_ENABLE_OAUTH_TEST=1 $(GO) test ./kook -run '^TestKOOKOAuthIntegration$$' -count=1 -v

# 生成根目录示例程序的本地二进制。
build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(PROJECT_NAME) .

# 验证 SDK 和所有示例包都能编译，不生成仓库内产物。
build-all:
	$(GO) build ./...

build-examples:
	mkdir -p $(BUILD_DIR)/examples
	@for example in $(EXAMPLES); do \
		echo "building examples/$$example"; \
		$(GO) build -o "$(BUILD_DIR)/examples/$$example" "./examples/$$example" || exit 1; \
	done

run:
	$(GO) run .

example-simple:
	$(GO) run ./examples/simple_bot

example-webhook:
	$(GO) run ./examples/webhook_bot

example-api:
	$(GO) run ./examples/api_usage

example-advanced:
	$(GO) run ./examples/advanced_bot

example-complete:
	$(GO) run ./examples/complete_api_demo

fmt:
	$(GO) fmt ./...

fmt-check:
	@files="$$($(GOFMT) -l .)"; \
	if [ -n "$$files" ]; then \
		echo "以下 Go 文件需要格式化："; \
		echo "$$files"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

clean:
	rm -rf $(BUILD_DIR)

install:
	$(GO) install .

docs:
	godoc -http=:6060

# 发布前的离线检查，不访问 KOOK。
verify: fmt-check test test-race vet lint vuln build-all

test-full: verify

help:
	@echo "可用目标："
	@echo "  deps              下载依赖，不修改依赖文件"
	@echo "  tidy              整理 go.mod 和 go.sum"
	@echo "  test              运行离线测试"
	@echo "  test-race         运行离线竞态测试"
	@echo "  test-integration  运行真实 KOOK 只读测试"
	@echo "  test-mutation     运行真实 KOOK 写入测试"
	@echo "  test-oauth        运行真实 OAuth 测试"
	@echo "  build             构建根目录程序到 build/"
	@echo "  build-all         编译 SDK 和所有示例"
	@echo "  build-examples    构建 5 个示例到 build/examples/"
	@echo "  run               运行根目录程序"
	@echo "  example-*         运行对应示例"
	@echo "  fmt               格式化 Go 代码"
	@echo "  fmt-check         检查 Go 代码格式"
	@echo "  vet               运行 go vet"
	@echo "  lint              运行 golangci-lint"
	@echo "  vuln              运行 govulncheck"
	@echo "  verify            执行发布前离线检查"
	@echo "  clean             删除 build/"
	@echo "  install           安装根目录程序"
	@echo "  docs              在 :6060 启动 godoc"
