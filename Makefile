BINARY=mridu
CMD=./cmd/mridu
GOBUILD=go build
GOTEST=go test
GOFLAGS=-ldflags="-s -w"

.PHONY: all build test test-race clean run run-all fmt lint

all: build

build:
	$(GOBUILD) $(GOFLAGS) -o $(BINARY) $(CMD)

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BINARY)-linux-amd64 $(CMD)

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BINARY)-darwin-amd64 $(CMD)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(GOFLAGS) -o $(BINARY)-darwin-arm64 $(CMD)

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o $(BINARY)-windows-amd64.exe $(CMD)

test:
	$(GOTEST) -v -count=1 ./lang/...

test-race:
	$(GOTEST) -v -race -count=1 ./lang/...

test-short:
	$(GOTEST) -v -count=1 -run 'TestSmoke' ./lang/...

test-ab:
	$(GOTEST) -v -count=1 -run 'TestAB' ./lang/...

test-integration:
	$(GOTEST) -v -count=1 -run 'TestIntegration' ./lang/...

test-existing:
	$(GOTEST) -v -count=1 -run 'TestLiterals|TestArithmetic|TestComparison|TestLogical|TestStringOps|TestVariables|TestScoping|TestControlFlow|TestFunctions|TestClosures|TestClasses|TestInheritance|TestNative|TestCompileErrors|TestRuntimeErrors|TestIntegration|TestEdgeCases' ./lang/...

test-exhaustive:
	$(GOTEST) -v -count=1 -run 'TestScanner|TestChunk|TestValue|TestObj|TestTypeCheck|TestClosure|TestCompileError|TestRuntimeError|TestOp|TestBoundary|TestStress|TestProperty|TestNegative|TestCross|TestFloat|TestGlobal|TestChained|TestCompiler' ./lang/...

clean:
	rm -f $(BINARY) $(BINARY)-*-*
	go clean ./...

run: build
	./$(BINARY) $(PROG)

run-all: build
	@for f in programs/*.mridu; do \
		echo "=== $$f ==="; \
		./$(BINARY) "$$f" 2>&1; \
		echo "---"; \
	done

fmt:
	go fmt ./...

lint:
	go vet ./...
