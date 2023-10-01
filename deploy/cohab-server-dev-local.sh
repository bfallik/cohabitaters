#!/bin/bash

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd "${SCRIPT_DIR}"

GOOGLE_APP_CREDENTIALS=$(< client_secret.json)
export GOOGLE_APP_CREDENTIALS

COOKIE_HASH_BLOCK_KEYS=$(< local-cookie-store-key.txt)
export COOKIE_HASH_BLOCK_KEYS

exec ../bin/cohab-server
