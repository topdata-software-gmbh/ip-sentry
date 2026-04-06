#!/bin/bash
#
# Pushes the Docker image to a registry
#
# Usage: ./push-docker-image.sh [-h|--help] [OPTIONS]
#   -h, --help     Show this help message
#   REGISTRY=...   Registry URL (default: docker.io)
#   IMAGE_NAME=... Image name (default: topdata/ip-sentry)
#   VERSION=...    Image version (default: latest)
#
# Example:
#   REGISTRY=ghcr.io VERSION=v1.0.0 ./push-docker-image.sh

set -e

usage() {
    echo "Usage: $0 [-h|--help] [OPTIONS]"
    echo "  -h, --help     Show this help message"
    echo "  REGISTRY=...   Registry URL (default: docker.io)"
    echo "  IMAGE_NAME=... Image name (default: topdata/ip-sentry)"
    echo "  VERSION=...    Image version (default: latest)"
    echo ""
    echo "Example:"
    echo "  REGISTRY=ghcr.io VERSION=v1.0.0 $0"
    exit 0
}

for arg in "$@"; do
    case "$arg" in
        -h|--help) usage ;;
    esac
done

REGISTRY="${REGISTRY:-docker.io}"
IMAGE_NAME="topdata/ip-sentry"
VERSION="${VERSION:-latest}"

FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${VERSION}"

echo "Building Docker image: ${FULL_IMAGE}"
docker build -t "${FULL_IMAGE}" .

echo "Pushing to registry: ${FULL_IMAGE}"
docker push "${FULL_IMAGE}"

echo "Done! Image pushed successfully."

