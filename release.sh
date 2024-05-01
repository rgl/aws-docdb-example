#!/bin/bash
set -euxo pipefail

APP_SOURCE_URL="https://github.com/$GITHUB_REPOSITORY"

if [[ "$GITHUB_REF" =~ \/v([0-9]+(\.[0-9]+)+(-.+)?) ]]; then
    APP_VERSION="${BASH_REMATCH[1]}"
else
    echo "ERROR: Unable to extract semver version from GITHUB_REF."
    exit 1
fi

APP_REVISION="$GITHUB_SHA"

IMAGE="ghcr.io/$GITHUB_REPOSITORY:$APP_VERSION"

docker build \
    --build-arg "APP_SOURCE_URL=$APP_SOURCE_URL" \
    --build-arg "APP_VERSION=$APP_VERSION" \
    --build-arg "APP_REVISION=$APP_REVISION" \
    -t "$IMAGE" \
    .

docker push "$IMAGE"
