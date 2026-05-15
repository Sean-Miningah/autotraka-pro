#!/usr/bin/env bash
set -euo pipefail

# Import the Meta Business API v23.0 OpenAPI schema into a Mockoon environment.
# Usage: ./scripts/import-mockoon-schema.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT="${REPO_ROOT}/mockoon/meta-mock.json"
SCHEMA_URL="https://raw.githubusercontent.com/facebook/openapi/main/business-messaging-api_v23.0.yaml"
SCHEMA_FILE="${REPO_ROOT}/mockoon/business-messaging-api_v23.0.yaml"
TMP_OUT="${REPO_ROOT}/mockoon/meta-mock.json.tmp"

echo "Downloading Meta Business API v23.0 OpenAPI schema..."
curl -sSL -o "${SCHEMA_FILE}" "${SCHEMA_URL}"
echo "Saved: ${SCHEMA_FILE}"

echo "Importing schema into Mockoon (this may take a minute)..."
# Mount the schema read-only and stream the JSON output from the container's /tmp
# so we avoid uid/gid permission mismatches with the host volume.
docker run --rm --entrypoint "" \
  -v "${REPO_ROOT}/mockoon:/schema:ro" \
  mockoon/cli:latest \
  sh -c 'mockoon-cli import -i /schema/business-messaging-api_v23.0.yaml -o /tmp/meta-mock.json.tmp --prettify && cat /tmp/meta-mock.json.tmp' \
  > "${TMP_OUT}"

mv "${TMP_OUT}" "${OUTPUT}"
echo "Generated: ${OUTPUT}"
