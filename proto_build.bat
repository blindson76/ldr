@echo off
protoc proto\loader.proto --go_out=. --go-grpc_out=.