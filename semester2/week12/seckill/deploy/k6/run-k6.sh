#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${NAMESPACE:-default}"
JOB_NAME="${JOB_NAME:-seckill-k6}"
SCRIPT_PATH="${SCRIPT_PATH:-deploy/k6/seckill.js}"
BASE_URL="${BASE_URL:-http://product:8000}"
USER_COUNT="${USER_COUNT:-50000}"
START_USER_ID="${START_USER_ID:-1}"
K6_STAGES="${K6_STAGES:-30s:20,1m:50,30s:0}"
HTTP_TIMEOUT="${HTTP_TIMEOUT:-5s}"
SLEEP_MS="${SLEEP_MS:-0}"

if [[ ! -f "${SCRIPT_PATH}" ]]; then
  echo "k6 script not found: ${SCRIPT_PATH}" >&2
  exit 1
fi

if [[ ! "${K6_STAGES}" =~ ^[0-9]+[smh]:[0-9]+(,[0-9]+[smh]:[0-9]+)*$ ]]; then
  echo "invalid K6_STAGES: ${K6_STAGES}" >&2
  echo "expected format: 30s:20,1m:50,30s:0" >&2
  exit 1
fi

kubectl -n "${NAMESPACE}" create configmap seckill-k6-script \
  --from-file=seckill.js="${SCRIPT_PATH}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NAMESPACE}" delete job "${JOB_NAME}" --ignore-not-found

cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: k6
          image: grafana/k6:latest
          args:
            - run
            - /scripts/seckill.js
          env:
            - name: BASE_URL
              value: "${BASE_URL}"
            - name: USER_COUNT
              value: "${USER_COUNT}"
            - name: START_USER_ID
              value: "${START_USER_ID}"
            - name: K6_STAGES
              value: "${K6_STAGES}"
            - name: HTTP_TIMEOUT
              value: "${HTTP_TIMEOUT}"
            - name: SLEEP_MS
              value: "${SLEEP_MS}"
          volumeMounts:
            - name: scripts
              mountPath: /scripts
      volumes:
        - name: scripts
          configMap:
            name: seckill-k6-script
EOF

while true; do
  complete_count="$(kubectl -n "${NAMESPACE}" get job "${JOB_NAME}" -o jsonpath='{.status.succeeded}' 2>/dev/null || true)"
  failed_count="$(kubectl -n "${NAMESPACE}" get job "${JOB_NAME}" -o jsonpath='{.status.failed}' 2>/dev/null || true)"
  pod_name="$(kubectl -n "${NAMESPACE}" get pod -l job-name="${JOB_NAME}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  pod_phase="$(kubectl -n "${NAMESPACE}" get pod -l job-name="${JOB_NAME}" -o jsonpath='{.items[0].status.phase}' 2>/dev/null || true)"

  echo "job=${JOB_NAME} pod=${pod_name:-N/A} phase=${pod_phase:-N/A} succeeded=${complete_count:-0} failed=${failed_count:-0}"

  if [[ "${complete_count:-0}" == "1" ]]; then
    kubectl -n "${NAMESPACE}" logs "${pod_name}"
    exit 0
  fi

  if [[ "${failed_count:-0}" != "" && "${failed_count:-0}" != "0" ]]; then
    kubectl -n "${NAMESPACE}" describe job "${JOB_NAME}" || true
    if [[ -n "${pod_name:-}" ]]; then
      kubectl -n "${NAMESPACE}" describe pod "${pod_name}" || true
      kubectl -n "${NAMESPACE}" logs "${pod_name}" || true
    fi
    exit 1
  fi

  sleep 3
done
