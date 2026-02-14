#!/usr/bin/env bash
set -euo pipefail

# Lint the canonical root spec.
redocly lint openapi-base.yml
