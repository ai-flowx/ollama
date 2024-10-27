#!/bin/bash

ldflags="-s -w"
target="ollama-serve"

go env -w GOPROXY=https://goproxy.cn,direct

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags "$ldflags" -o bin/$target main.go
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -ldflags "$ldflags" -o bin/$target.exe main.go
