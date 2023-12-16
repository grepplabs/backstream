FROM golang:1.21-alpine3.18 AS builder
RUN apk add --no-cache alpine-sdk ca-certificates

WORKDIR /app
COPY . /app

WORKDIR /app/tests/backstream/http
RUN go mod download
COPY *.go ./
RUN go build -o /main

FROM alpine:3.18
COPY --from=builder /main /main
EXPOSE 8080
ENTRYPOINT ["/main"]
