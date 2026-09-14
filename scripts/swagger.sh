#!/bin/sh
set -eu

# execute this first
# go install github.com/swaggo/swag/cmd/swag@latest

(
	cd backend
	swag init -o docs --parseDependency --parseInternal
)
cp backend/docs/swagger.json backend/docs/swagger.yaml docs/
