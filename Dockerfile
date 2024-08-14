FROM golang:1.22 AS builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./... 

FROM debian:12.5-slim
COPY --from=builder /usr/local/bin/app /usr/local/bin
COPY logo.jpg /usr/local/bin
WORKDIR /usr/local/bin

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates
RUN update-ca-certificates

ENTRYPOINT ["/usr/local/bin/app"]

