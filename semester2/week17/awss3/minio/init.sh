#!/bin/sh
# Wait for minIO to be reachable, then create the bucket used by the demo.
# Designed to run inside the `minio/mc` image as a one-shot init container.
set -eu

MC=/usr/bin/mc
ALIAS=local
BUCKET=${S3_BUCKET:-test-bucket}
ENDPOINT=${S3_ENDPOINT:-http://minio:9000}
ACCESS_KEY=${MINIO_ROOT_USER:-minioadmin}
SECRET_KEY=${MINIO_ROOT_PASSWORD:-minioadmin}

echo ">> waiting for minIO at ${ENDPOINT}..."
for i in $(seq 1 60); do
    if ${MC} --quiet alias set "${ALIAS}" "${ENDPOINT}" "${ACCESS_KEY}" "${SECRET_KEY}" 2>/dev/null; then
        if ${MC} --quiet ready "${ALIAS}" 2>/dev/null; then
            break
        fi
    fi
    sleep 2
done

echo ">> configuring alias ${ALIAS} -> ${ENDPOINT}"
${MC} alias set "${ALIAS}" "${ENDPOINT}" "${ACCESS_KEY}" "${SECRET_KEY}"

echo ">> ensuring bucket ${BUCKET} exists"
if ! ${MC} ls "${ALIAS}/${BUCKET}" >/dev/null 2>&1; then
    ${MC} mb -p "${ALIAS}/${BUCKET}"
else
    echo "   already present"
fi

echo ">> setting anonymous download (handy for curl-based smoke tests)"
${MC} anonymous set download "${ALIAS}/${BUCKET}" || true

echo ">> ready"
