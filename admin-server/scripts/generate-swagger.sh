#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -st -g main.go -o docs --parseDependency --parseInternal
