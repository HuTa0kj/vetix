#!/bin/bash

# macOS Intel (x86_64)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ./vetix-mac-amd64 ./cmd/vetix

# macOS Apple Silicon (arm64)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ./vetix-mac-arm64 ./cmd/vetix

# Linux 64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./vetix-linux ./cmd/vetix

# Windows 64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ./vetix-windows.exe ./cmd/vetix

echo "Build completed."
