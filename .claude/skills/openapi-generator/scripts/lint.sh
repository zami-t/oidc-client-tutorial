#!/usr/bin/env bash
set -euo pipefail

redocly lint --config .redocly.yaml openapi/openapi.yml
