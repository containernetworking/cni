#!/bin/bash
protoc -I pkg/types/v2/ \
    -I${GOPATH}/src/github.com/protocolbuffers/protobuf/src/ \
    --go_out=pkg/types/v2/ --go_opt=paths=source_relative \
    --go-grpc_out=pkg/types/v2/ --go-grpc_opt=paths=source_relative \
    pkg/types/v2/cni.proto
