BUILD_DATE=$(shell git log -1 --format="%cd" --date=short | sed s/-//g)
BUILD_NUM=$(shell  git rev-list --count HEAD)
BUILD_HASH=$(shell git rev-parse --short HEAD)

LDFLAGS="-X main.version=${BUILD_DATE}.${BUILD_NUM}_${BUILD_HASH}-1"
GCFLAGS="-trimpath ${GOPATH}/src"

build:
	go build -x -ldflags=${LDFLAGS} -gcflags ${GCFLAGS} .

man:
	@ronn -r man.markdown

clean:
	@git clean -ffdx

test:
	tests/run_tests
