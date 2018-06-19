.PHONY: install build test lint fmt clean

install: fmt test
	go install

build: fmt test
	go build

test:
	go test -race
	cd test && bundle install --frozen && ./end-to-end -v

lint:
	golint

fmt:
	! gofmt -d -e -s *.go 2>&1 | tee /dev/tty | read

clean:
	go clean
