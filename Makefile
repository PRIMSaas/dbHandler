
all: fmt tidy build test vet 

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

deploy:
	docker build -t drjimdb .
	docker tag drjimdb australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb
	docker push australia-southeast2-docker.pkg.dev/drjim-f2087/drjimrepo/drjimdb

.PHONY: all clean it fmt tidy build test vet lint cover

clean:
	rm -f *.o dbHandler c.out
