PRJ_NAME=Minder
AUTHOR="Meteormin \(miniyu97@gmail.com\)"
PRJ_BASE=$(shell pwd)
PRJ_DESC=$(PRJ_NAME) Deployment and Development Makefile.\n Author: $(AUTHOR)

mod ?= ""

# OS와 ARCH가 정의되어 있지 않으면 기본값을 설정합니다.
# uname -s는 OS 이름(예: Linux, Darwin 등)을 반환하고, tr를 통해 소문자로 변환합니다.
OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
# 아키텍처 정보를 반환합니다. (예: amd64, arm64 등)
ARCH := $(shell ./scripts/detect-arch.sh)

.DEFAULT: help
.SILENT:;

##help: helps (default)
.PHONY: help
help: Makefile
	echo ""
	echo " $(PRJ_DESC)"
	echo ""
	echo " Usage:"
	echo ""
	echo "	make {command}"
	echo ""
	echo " Commands:"
	echo ""
	sed -n 's/^##/	/p' $< | column -t -s ':' |  sed -e 's/^/ /'
	echo ""

##install
.PHONY: install
install:
	@echo "[install] install and initalize"
	./devtools/install.sh
	go mod download
	cd ./webui && yarn install

##test report={[0=inactive, 1=active]}: test
.PHONY: test
test:
ifeq ($(report), 1)
	@echo "[test] go test with report"
	mkdir -p reports
	go test -v -coverprofile=reports/coverage.out ./... > reports/test.out
	go tool cover -html=reports/coverage.out -o reports/coverage.html
else
	@echo "[test] go test"
	go test ./...
endif

##build os={os [linux, darwin]} arch={arch [amd64, arm64]} mod={entrypoint}: build application for cross compile
.PHONY: build
build: os ?= $(OS)
build: arch ?= $(ARCH)
build: mod ?= ""
build:
	@echo "[build] Building $(mod) for $(os)/$(arch)"
ifeq ($(os), linux)
	CC=$(cc) CGO_ENABLED=1 GOOS=$(os) GOARCH=$(arch) go build $(LDFLAGS) -o build/$(mod)-$(os)-$(arch) ./cmd/$(mod)/main.go
else
	CC=$(cc) CGO_ENABLED=1 GOOS=$(os) GOARCH=$(arch) go build -o build/$(mod)-$(os)-$(arch) ./cmd/$(mod)/main.go
endif

##clean: clean application
.PHONY: clean
clean:
	@echo "[clean] Cleaning build directory"
	rm -rf build/*

##run mod={entrypoint} flags={flags}: run application
.PHONY: run
run: mod ?= ""
run: flags ?= ""
run:
	@echo "[run] running application"
	@echo "mod: $(mod)"
	@echo "flags: $(flags)"
	go run ./cmd/$(mod)/main.go $(flags)