#!/usr/bin/env bash
set -ex

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
cd "$SCRIPT_DIR"

: ${CRYPTHOPPER_KEY:=/tmp/key}
: ${CRYPTHOPPER_DATA_DIR:="polygon-mumbai-$RANDOM"}
: ${CRYPTHOPPER_NODE_ADDRESS:=https://rpc-mumbai.matic.today}

echo "Testing with db in $CRYPTHOPPER_DATA_DIR"

rm -rf "$CRYPTHOPPER_DATA_DIR" || true

payouts import --data-dir "$CRYPTHOPPER_DATA_DIR" test-payouts-small.csv

#Infura ID is personal test project of Marton Elek, can be replaced in real test environment
payouts run --contract 0xfe4f5145f6e09952a5ba9e956ed0c25e3fa4c7f1 --node-address=$CRYPTHOPPER_NODE_ADDRESS --chain-id 80001 ${CRYPTHOPPER_DATA_DIR}/test-payouts-small $CRYPTHOPPER_KEY

payouts audit --node-address=$CRYPTHOPPER_NODE_ADDRESS --chain-id 80001 --data-dir ${CRYPTHOPPER_DATA_DIR} test-payouts-small.csv
