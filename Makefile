
all: fmt tidy vend build test vet lint cover

it: all

fmt:
	go fmt

tidy:
	go mod tidy

vend:
	go mod vendor

build:
	rm -f dbHandler 
	go build

test:
	go test -coverprofile=c.out

vet:
	go vet

lint:
	golangci-lint run --disable typecheck --disable unused 

cover:
	go tool cover -func=c.out

.PHONY: all clean it fmt tidy build test vet lint cover

clean:
	rm -f *.o crex
