GOCMD=CGO_ENABLED=0 go
GO_BUILD_FLAGS=-trimpath -ldflags '-s -w -extldflags "-static"' -mod vendor
BIN=sigplot_data_service
IMAGE=sds:0.7
FUZZTIME ?= 10s

.PHONY: all ui sds sds-ui docker \
	fmt vet lint test test-race test-fuzz check clean

# --- Build targets ---

all: ui sds-ui

ui:
	npm --prefix ./ui/webapp ci
	npm --prefix ./ui/webapp run build

sds:
	$(GOCMD) build $(GO_BUILD_FLAGS) -o $(BIN) ./cmd/sds

sds-ui: ui
	$(GOCMD) build $(GO_BUILD_FLAGS) -tags ui -o $(BIN) ./cmd/sds

docker:
	docker build -t $(IMAGE) .

# --- Code quality targets ---

fmt:
	@echo "==> Running gofmt..."
	@gofmt -l -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	@echo "==> Running go vet..."
	go vet ./...

lint:
	@echo "==> Running golangci-lint..."
	golangci-lint run ./...

# --- Test targets ---

test:
	@echo "==> Running tests..."
	go test ./... -count=1

test-race:
	@echo "==> Running tests with race detector..."
	go test ./... -race -count=1

test-fuzz:
	@echo "==> Running fuzz tests ($(FUZZTIME) each)..."
	go test ./internal/bluefile -run=^$$ -fuzz=^FuzzConvertFileData$$ -fuzztime=$(FUZZTIME)
	go test ./internal/image -run=^$$ -fuzz=^FuzzTransform$$ -fuzztime=$(FUZZTIME)

# --- Combined targets ---

check: fmt vet lint test
	@echo "==> All checks passed."

clean:
	@rm -f $(BIN)
	@rm -rf sdscache/
	@rm -rf ui/webapp/dist/
