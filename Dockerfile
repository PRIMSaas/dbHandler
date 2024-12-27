FROM golang:1.23.2 AS builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/app ./... 

FROM gcr.io/distroless/static-debian12
COPY --from=builder /usr/local/bin/app /
WORKDIR /

ENTRYPOINT ["/app"]

