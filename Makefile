APP_NAME := fingrab
DOCKER_IMAGE := hallyg/${APP_NAME}

GIT_REF := $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short=8 --verify HEAD)
BUILD_VERSION ?= $(GIT_REF)
BUILD_SHA := $(shell git rev-parse --verify HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
PWD := $(shell pwd)
BUILD_DIR := ${PWD}/build

GO_CMD ?= go
GO_LDFLAGS ?= -s -w -buildid= -X 'github.com/HallyG/fingrab/cmd/fingrab/root.BuildShortSHA=$(BUILD_VERSION)'
GO_PKG_MAIN := ${PWD}/main.go
GO_PKGS := $(PWD)/internal/... $(PWD)/cmd/fingrab/... ${PWD}/main.go
GO_COVERAGE_FILE := $(BUILD_DIR)/cover.out
GO_COVERAGE_TEXT_FILE := $(BUILD_DIR)/cover.txt
GO_COVERAGE_HTML_FILE := $(BUILD_DIR)/cover.html
GOLANGCI_CMD := go tool golangci-lint
GOLANGCI_ARGS ?= --fix --concurrency=4
GOLANGCI_FILES ?= ${GO_PKGS}

DOCKER_DIR := ${PWD}
DOCKER_FILE := ${DOCKER_DIR}/Dockerfile

.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## clean: remove build artifacts and temporary files
.PHONY: clean
clean:
	@rm -rf ${BUILD_DIR}
	@$(GO_CMD) clean

## audit: format, vet, and lint Go code
.PHONY: audit
audit: clean
	@$(GO_CMD) mod tidy
	@$(GO_CMD) mod verify
	@$(GO_CMD) fmt ${GO_PKGS}
	@$(GO_CMD) vet ${GO_PKGS}
	@${GOLANGCI_CMD} run ${GOLANGCI_ARGS} ${GOLANGCI_FILES}

## test: run tests
.PHONY: test
test:
	@$(GO_CMD) test -race $(if $(VERBOSE),-v) ${GO_PKGS}

## test/cover: run tests with coverage
.PHONY: test/cover
test/cover:
	@mkdir -p ${BUILD_DIR}
	@rm -f ${GO_COVERAGE_FILE} ${GO_COVERAGE_TEXT_FILE} ${GO_COVERAGE_HTML_FILE}
	@$(GO_CMD) test -race -coverprofile=${GO_COVERAGE_FILE} ${GO_PKGS}
	@$(GO_CMD) tool cover -func ${GO_COVERAGE_FILE} -o ${GO_COVERAGE_TEXT_FILE}
	@$(GO_CMD) tool cover -html ${GO_COVERAGE_FILE} -o ${GO_COVERAGE_HTML_FILE}

## docker/build: build the application docker image
.PHONY: docker/build
docker/build:
	@docker build \
		-t ${DOCKER_IMAGE}:$(BUILD_VERSION) \
		-t ${DOCKER_IMAGE}:latest \
		-f $(DOCKER_FILE) ${PWD} \
		--build-arg BUILD_DATE=${BUILD_DATE} \
		--build-arg COMMIT_HASH=${BUILD_SHA} \
		--build-arg BUILD_VERSION=${BUILD_VERSION} \
		--build-arg GO_LDFLAGS="${GO_LDFLAGS}"

## docker/run: run the application docker image
.PHONY: docker/run
docker/run: docker/build
	@docker run --rm --name ${APP_NAME} -t ${DOCKER_IMAGE}:latest

## build: build the application
.PHONY: build
build:
	@echo "GO_LDFLAGS: $(GO_LDFLAGS)"
	@$(GO_CMD) build -o ${BUILD_DIR}/${APP_NAME} -trimpath -mod=readonly -ldflags="$(GO_LDFLAGS)" ${GO_PKG_MAIN}

## run: run the application	
.PHONY: run
run: build
	@${BUILD_DIR}/${APP_NAME} version