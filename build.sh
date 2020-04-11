#!/usr/local/bin/bash

go mod download

GOOS=linux go build -o flyenv-linux
GOOS=darwin go build -o flyenv-darwin
GOOS=windows go build -o flyenv-windows.exe