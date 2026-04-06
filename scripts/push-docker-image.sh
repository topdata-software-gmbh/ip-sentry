#!/bin/bash

set -e

REGISTRY="${REGISTRY:-docker.io}"
IMAGE_NAME="${IMAGE_NAME:-ip-sentry}"
VERSION="${VERSION:-latest}"

FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${VERSION}"

echo "Building Docker image: ${FULL_IMAGE}"
docker build -t "${FULL_IMAGE}" .

echo "Pushing to registry: ${FULL_IMAGE}"
docker push "${FULL_IMAGE}"

echo "Done! Image pushed successfully."
