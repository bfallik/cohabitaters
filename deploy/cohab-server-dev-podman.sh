#!/bin/bash

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

GOOGLE_APP_CREDENTIALS=$(< "${SCRIPT_DIR}"/client_secret.json)
export GOOGLE_APP_CREDENTIALS

exec podman run -p 8080:8080 -t cohab-server:latest .
