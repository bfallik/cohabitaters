#!/bin/bash

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

GOOGLE_APP_CREDENTIALS=$(< "${SCRIPT_DIR}"/client_secret.json)
export GOOGLE_APP_CREDENTIALS

COOKIE_STORE_KEY=$(< "${SCRIPT_DIR}"/local-cookie-store-key.txt)
export COOKIE_STORE_KEY

exec "${SCRIPT_DIR}/../bin/cohab-server"
