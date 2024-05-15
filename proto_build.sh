#!/bin/sh

protoc --experimental_allow_proto3_optional proto/loader.proto --go_out=. --go-grpc_out=.
