#!/bin/bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o out/agit-arm64 main.go

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o out/agit-x86 main.go

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o out/agit.exe main.go

#./main

echo 'build success'