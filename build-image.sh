#!/bin/bash
set -e

REGISTRY_USER="jetfuls"
APP_VERSION="1.0"
# ROCKY_VERSIONS=("9")
# ROCKY_VERSIONS=("8")
ROCKY_VERSIONS=("10")
#ROCKY_VERSIONS=("8" "9" "10")
ALIYUN_REGISTRY="crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com"

for ROCKY_VERSION in "${ROCKY_VERSIONS[@]}"; do
    IMAGE_NAME="${REGISTRY_USER}/kickcraft:${APP_VERSION}-rocky${ROCKY_VERSION}"
    ALIYUN_IMAGE="${ALIYUN_REGISTRY}/${REGISTRY_USER}/kickcraft:${APP_VERSION}-rocky${ROCKY_VERSION}"

    echo "=== Building Docker image: $IMAGE_NAME ==="
    make docker-build REGISTRY_USER="$REGISTRY_USER" APP_VERSION="$APP_VERSION" ROCKY_VERSION="$ROCKY_VERSION"

    echo "=== Pushing to Docker Hub: $IMAGE_NAME ==="
    docker push "$IMAGE_NAME"

    echo "=== Tagging for Aliyun: $ALIYUN_IMAGE ==="
    docker tag "$IMAGE_NAME" "$ALIYUN_IMAGE"

    echo "=== Pushing to Aliyun: $ALIYUN_IMAGE ==="
    docker push "$ALIYUN_IMAGE"

    echo "=== Done: Rocky $ROCKY_VERSION ==="
done

echo "All images built and pushed successfully!"
