SOURCEDIR := .
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

VERSION_GIT := $(shell sh -c 'git describe --always --tags --abbrev=7')
BUILD_TIME := `date +%FT%T%z`
LDFLAGS := -ldflags "-X main.Version=${VERSION_GIT} -X main.BuildTime=${BUILD_TIME}"

s3-cli: $(SRC)
	GOARCH=amd64 GOOS=linux go build ${LDFLAGS} -o $@ $(SOURCES)

clean: $(SRC)
	rm -f s3-cli

test:
	go test
