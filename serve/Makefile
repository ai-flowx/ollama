# Config

# Build

.PHONY: FORCE

build: go-build
.PHONY: build

clean: go-clean
.PHONY: clean

lint: go-lint
.PHONY: lint

test: go-test
.PHONY: test

all-test: go-all-test
.PHONY: all-test


# Non-PHONY targets (real files)

go-build: FORCE
	./script/build.sh

go-clean: FORCE
	./script/clean.sh

go-lint: FORCE
	./script/lint.sh

go-test: FORCE
	./script/test.sh report

go-all-test: FORCE
	./script/test.sh all
