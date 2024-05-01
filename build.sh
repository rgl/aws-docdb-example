#!/bin/bash
set -euxo pipefail

APP_REVISION="${GITHUB_SHA:-0000000000000000000000000000000000000000}"

docker build \
    --build-arg "APP_REVISION=$APP_REVISION" \
    .
