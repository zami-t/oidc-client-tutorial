#!/usr/bin/env bash
set -euo pipefail

# Split OpenAPI into ./openapi directory.
redocly split openapi-base.yml --outDir=openapi
