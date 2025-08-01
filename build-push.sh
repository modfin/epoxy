#!/usr/bin/env bash

VERSION=$(date +%Y-%m-%dT%H.%M.%S)-$(git log -1 --pretty=format:"%h")
NAME=epoxy
IMAGE_NAME=modfin/${NAME}

BUILDER=$(docker buildx create) || exit 1

docker buildx build --builder=${BUILDER} \
    --push \
    --target epoxy \
    --platform linux/amd64,linux/arm64 \
    -f ./Dockerfile \
    -t ${IMAGE_NAME}:latest \
    -t ${IMAGE_NAME}:${VERSION} \
    .

docker buildx build --builder=${BUILDER} \
    --push \
    --target epoxy-slim \
    --platform linux/amd64,linux/arm64 \
    -f ./Dockerfile \
    -t ${IMAGE_NAME}-slim:latest \
    -t ${IMAGE_NAME}-slim:${VERSION} \
    .

docker buildx rm "${BUILDER}"
