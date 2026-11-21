.PHONY: build run test clean dist install all

BINARY := picox
VERSION := 0.1.0

# 默认：编译出完全独立的静态二进制（无外部依赖，可直接拷贝到任何同架构 Linux 运行）
build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) .

# 完全静态链接（连 libc 都不依赖，可在任何 Linux 上裸跑）
static:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) .

run: build
	./$(BINARY)

repl: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

# 交叉编译：生成多平台独立二进制到 dist/
dist:
	@mkdir -p dist
	@echo "=== 编译多平台独立二进制 ==="
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/picox-windows-arm64.exe .
	@echo "=== 完成，产物在 dist/ ==="
	@ls -lh dist/

# 安装到系统 PATH（默认 /usr/local/bin）
install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)

# 运行示例
hello: build
	./$(BINARY) run examples/hello.px

fib: build
	./$(BINARY) run examples/fibonacci.px

oop: build
	./$(BINARY) run examples/oop.px

.PHONY: hello fib oop
