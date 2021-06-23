# Based on https://betterprogramming.pub/my-ultimate-makefile-for-golang-projects-fcc8ca20c9bb
# Extended with knowledge from https://vic.demuzere.be/articles/golang-makefile-crosscompile/

GOVERSION ?= 1.16 ## Set the Go version for Docker
GOCMD := go
GOFMT := $(GOCMD) fmt
GOMOD := $(GOCMD) mod
GOTEST := $(GOCMD) test
GOBUILD := $(GOCMD) build
CGO ?= 1 ## Enable / disable CGO support
GOMODULE ?= on ## Enable / disable Go module support
BINARY_NAME ?= VlanLister ## Set the name of the resulting binary
VERSION ?= $(shell awk 'BEGIN {FS = "\\042"} /(t|T)oolVersion.+=/ {printf "%s", $$2; exit}' *.go) ## Set the version for release
COMMIT_ID := $(word 1, $(shell git log --oneline -n1))
EXPORT_RESULT ?= true ## Export the result of the linter
OUT_DIR ?= out ## The base out dir for all results
USE_DOCKER ?= ## Use Docker for all targets

PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 ## Platforms to compile for with target release

RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY := help fmt-go fmt lint-go lint pretty test build $(PLATFORMS) release all clean
.DEFAULT_GOAL := help

sgoversion = $(strip $(GOVERSION))
scgo = $(strip $(CGO))
sgomodule = $(strip $(GOMODULE))
sbinary_name = $(strip $(BINARY_NAME))
sversion = $(strip $(VERSION))
sexport_result = $(strip $(EXPORT_RESULT))
sout_dir = $(strip $(OUT_DIR))
suse_docker = $(strip $(USE_DOCKER))

platformtemp = $(subst /, ,$@)
os = $(word 1, $(platformtemp))
arch = $(word 2, $(platformtemp))
os_windows = $(findstring $(os), "windows")
release_dir = $(sbinary_name)_$(sversion)
binary_filename = $(sbinary_name)_$(COMMIT_ID)
binary_filename_version = $(sbinary_name)_$(sversion)_$(COMMIT_ID)_$(os)-$(arch)
final_filename = $(if $(os_windows),$(addsuffix .exe,$(binary_filename)),$(binary_filename))
final_filename_version = $(if $(os_windows),$(addsuffix .exe,$(binary_filename_version)),$(binary_filename_version))

help: ## Show this help
	@echo ''
	@echo 'Usage:'
	@echo '  [${CYAN}VARIABLE=value${RESET} ...] ${GREEN}make${RESET} ${YELLOW}target${RESET}'
	@echo ''
	@echo 'Variables:'
	@awk 'BEGIN {FS = "\\077=.*?## "} /^[A-Z_-]+ *\077= *.+/ {printf "  ${CYAN}%-16s${RESET}%s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${YELLOW}%-16s${RESET}%s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ''

tidy: ## Run go mod tidy
ifneq (,$(suse_docker))
	docker run --rm -v $(shell pwd):/app -w /app golang:$(sgoversion) $(GOMOD) tidy
else
	$(GOMOD) tidy
endif

fmt-go: ## Run gofmt on the project
ifneq (,$(suse_docker))
	docker run --rm -v $(shell pwd):/app -w /app golang:$(sgoversion) $(GOFMT) ./...
else
	$(GOFMT) ./...
endif

fmt: fmt-go ## Run all available formatters

lint-go: ## Use golintci-lint on your project
ifneq (,$(suse_docker))
	mkdir -p $(sout_dir)
	$(eval OUTPUT_OPTIONS = $(shell [ "${sexport_result}" == "true" ] && echo "--out-format checkstyle ./... | tee /dev/tty > ${sout_dir}/checkstyle-report.xml" || echo "" ))
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=65s $(OUTPUT_OPTIONS)
else
	@echo "${RED}Error:${RESET} Target lint-go is not available without Docker."
endif

lint: lint-go ## Run all available linters

pretty: tidy fmt lint ## Run targets tidy, fmt and lint

test: ## Run the tests of the project
ifneq (,$(suse_docker))
	docker run --rm -v $(shell pwd):/app -w /app golang:$(sgoversion) $(GOTEST) -race -cover ./...
else
	$(GOTEST) -race -cover ./...
endif

build: ## Build an executable
	mkdir -p $(sout_dir)/bin
ifneq (,$(suse_docker))
	docker run --rm -v $(shell pwd):/app -w /app -e CGO_ENABLED=$(scgo) -e GO111MODULE=$(sgomodule) golang:$(sgoversion) $(GOBUILD) -o $(sout_dir)/bin/$(final_filename) .
else
	CGO_ENABLED=$(scgo) GO111MODULE=$(sgomodule) $(GOBUILD) -o $(sout_dir)/bin/$(final_filename) .
endif

$(PLATFORMS): ## Build a platform specific release
	mkdir -p $(sout_dir)/$(release_dir)
ifneq (,$(suse_docker))
	docker run --rm -v $(shell pwd):/app -w /app -e CGO_ENABLED=$(scgo) -e GO111MODULE=$(sgomodule) -e GOOS=$(os) -e GOARCH=$(arch) golang:$(sgoversion) $(GOBUILD) -o $(sout_dir)/$(release_dir)/$(final_filename_version) .
else
	CGO_ENABLED=$(scgo) GO111MODULE=$(sgomodule) GOOS=$(os) GOARCH=$(arch) $(GOBUILD) -o $(sout_dir)/$(release_dir)/$(final_filename_version) .
endif

release: $(PLATFORMS) ## Build a complete release

all: pretty test release ## Run targets pretty, test and release

clean: ## Remove build related files
	rm -rf $(sout_dir)
