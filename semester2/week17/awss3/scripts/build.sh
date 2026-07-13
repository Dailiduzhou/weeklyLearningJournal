#!/usr/bin/env bash
# Build the project with vcpkg. Mirrors the steps run inside the Dockerfile
# so that local development and CI use the same flow.
set -euo pipefail

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${PROJECT_ROOT}/build"
VCPKG_ROOT="${VCPKG_ROOT:-${PROJECT_ROOT}/vcpkg}"

if [[ ! -x "${VCPKG_ROOT}/vcpkg" ]]; then
    echo ">> bootstrapping vcpkg"
    "${VCPKG_ROOT}/bootstrap-vcpkg.sh"
fi

mkdir -p "${BUILD_DIR}"
cmake -S "${PROJECT_ROOT}" -B "${BUILD_DIR}" \
      -DCMAKE_TOOLCHAIN_FILE="${VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake" \
      "$@"

cmake --build "${BUILD_DIR}" -j "$(nproc)"
echo ">> build artifacts in ${BUILD_DIR}/awss3"
