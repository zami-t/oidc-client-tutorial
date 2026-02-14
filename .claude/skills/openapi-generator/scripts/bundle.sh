#!/usr/bin/env bash
set -euo pipefail

redocly bundle openapi/openapi.yml -o openapi/openapi.bundle.yaml
