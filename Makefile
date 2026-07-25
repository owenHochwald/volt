APP_EXECUTABLE=volt
INSTALL_PATH=/usr/local/bin

build:
	go build -o ${APP_EXECUTABLE} ./cmd/volt/main.go

build-mac:
	GOARCH=amd64 GOOS=darwin go build -o ${APP_EXECUTABLE} ./cmd/volt/main.go

install: build
	sudo cp ${APP_EXECUTABLE} ${INSTALL_PATH}/

uninstall:
	sudo rm -f ${INSTALL_PATH}/${APP_EXECUTABLE}

run: build
	./${APP_EXECUTABLE}

clean:
	go clean
	rm -f ${APP_EXECUTABLE}

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

benchmark-comparison:
	@./benchmarks/comparison/run.sh

benchmark-smoke:
	@BENCH_CONCURRENCY=2 BENCH_COOLDOWN=0 BENCH_DURATION=1s BENCH_RUNS=1 BENCH_WARMUP_DURATION=1s ./benchmarks/comparison/run.sh

demo:
	@./scripts/record-demo.sh

.PHONY: build build-mac install uninstall run clean test test-coverage benchmark-comparison benchmark-smoke demo
