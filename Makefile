all: test

test:
	@mkdir -p fixtures
	go get .
	go test -cover