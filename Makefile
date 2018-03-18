# project name
PROJECT=gocrawler
VERSION=0.1.1

# Log Prefixes
INFO=[INFO]
ERRR=[ERRR]
WARN=[WARN]

# all targets
all: test pkg

# run tests
test:
	@echo "$(INFO) executing tests"
	@go test ./... || exit 1
	@echo "$(INFO) successfully executed tests"

# package artifact
pkg: test
	@echo "$(INFO) packaging binary"
	@CGO_ENABLED=0 \
	go build -a \
	    --ldflags "-s -w -extld ld -extldflags -static" -o $(PROJECT)

.PHONY: all test pkg
