#!/usr/bin/env bash
set -ex

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
cd "$SCRIPT_DIR"

: ${CRYPTHOPPER_KEY:=/tmp/key}
: ${CRYPTHOPPER_DATA_DIR:="zksync-rinkeby-$RANDOM"}

echo "Testing with db in $CRYPTHOPPER_DATA_DIR"

rm -rf "$CRYPTHOPPER_DATA_DIR" || true

payouts import --data-dir "$CRYPTHOPPER_DATA_DIR" test-payouts.csv

payouts run --type=zksync --node-address=https://rinkeby-api.zksync.io/ --chain-id 4 ${CRYPTHOPPER_DATA_DIR}/test-payouts $CRYPTHOPPER_KEY

payouts audit --type=zksync --node-address=https://rinkeby-api.zksync.io --chain-id 4 --data-dir ${CRYPTHOPPER_DATA_DIR} test-payouts.csv
