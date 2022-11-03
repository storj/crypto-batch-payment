#!/usr/bin/env bash
set -ex

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
cd "$SCRIPT_DIR"

: ${CRYPTHOPPER_KEY:=/tmp/key}
: ${CRYPTHOPPER_DATA_DIR:="eth-rinkeby-$RANDOM"}

echo "Testing with db in $CRYPTHOPPER_DATA_DIR"

rm -rf "$CRYPTHOPPER_DATA_DIR" || true

payouts import --data-dir "$CRYPTHOPPER_DATA_DIR" test-payouts.csv

#Infura ID is personal test project of Marton Elek, can be replaced in real test environment
payouts run --contract 0x8098165d982765097E4aa17138816e5b95f9fDb5 --node-address=$CRYPTHOPPER_NODE_ADDRESS --chain-id 4 ${CRYPTHOPPER_DATA_DIR}/test-payouts $CRYPTHOPPER_KEY

payouts audit --node-address=$CRYPTHOPPER_NODE_ADDRESS --chain-id 4 --data-dir ${CRYPTHOPPER_DATA_DIR} test-payouts.csv
