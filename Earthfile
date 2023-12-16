VERSION 0.7
FROM golang:1.21-alpine3.19


generate:
    FROM +sources
    RUN protoc -I=./ --go_out=module=github.com/grepplabs/backstream/internal/message:internal/message ./internal/proto/message.proto
    SAVE ARTIFACT internal/message/message.pb.go AS LOCAL internal/message/message.pb.go

sources:
    ARG PROTOC_GO_VERSION=v1.31.0
    ARG PROTOC_REL="https://github.com/protocolbuffers/protobuf/releases"
    ARG PROTOC_VERSION=25.1

    RUN apk add curl
    RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GO_VERSION}
    RUN curl -LO ${PROTOC_REL}/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
    RUN unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local
    COPY . .
