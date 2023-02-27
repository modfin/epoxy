#!/usr/bin/env bash

VERSION=$(date +%Y-%m-%dT%H.%M.%S)-$(git log -1 --pretty=format:"%h")
NAME=epoxy
IMAGE_NAME=modfin/${NAME}

docker build -f ./Dockerfile \
    -t ${IMAGE_NAME}:latest \
    -t ${IMAGE_NAME}:${VERSION} \
    . || exit 1

docker push ${IMAGE_NAME}:latest || exit 1
docker push ${IMAGE_NAME}:${VERSION} || exit 1
