#!/bin/bash

GOOS=darwin GOARCH=amd64 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-macOS64 .
#GOOS=darwin GOARCH=386 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-macOS32 .

GOOS=windows GOARCH=amd64 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-Windows64.exe .
GOOS=windows GOARCH=386 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-Windows32.exe .

GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-Linux64 .
GOOS=linux GOARCH=386 go build -ldflags="-extldflags=-static -w -s" -o ./bin/MetaOrganize-Linux32 .