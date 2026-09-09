#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -st -g doc.go -d internal/miniapp,pkg/response -o docs-miniapp --parseDependency --parseInternal --instanceName miniapp
cp docs-miniapp/miniapp_swagger.json docs-miniapp/swagger.json
cp docs-miniapp/miniapp_swagger.yaml docs-miniapp/swagger.yaml
