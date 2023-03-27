# Crypto Batch Payment

A high volume ERC20 token payment application.

Storj has been at the forefront of making token payments on Ethereum at scale. We've built and refined tools along the way and incorporated innovations like payments on L2 via zkSync and Polygon.

While we've been busy making Storj the best decentralized storage service in the world and operating our own Satellites, we know that addressing payments at scale is an important function of a satellite and that anyone operating a satellite would benefit from access to this code.

While we use this tool in production and we're making it available for use by anyone, you can use it only at your own risk. We will not provide support for the code in this repository.

This software, crypthopper-go, is designed to send bulk transactions on Ethereum for a variety of use cases. Like any software that interacts with wallets, crypto and the blockchain, it is easy to make mistakes and mistakes are permanent.

**Please keep in mind that improper usage or unexpected bugs may cause token loss.** In addition, you should be mindful of how this code is used, where it is run and where it is committed, as the private keys are imported into the code while in use.

**Storj is not legally responsible for any outcomes based on your use of this code.** The material embodied in this software is provided to you "as-is" and without warranty of any kind, express, implied or otherwise, including without limitation, any warranty of fitness for a particular purpose. In no event shall Storj Labs International SEZC or Storj Inc be liable to you or anyone else for any direct, special, incidental, indirect or consequential damages of any kind, or any damages whatsoever, including without limitation, loss of profit, loss of use, savings or revenue, or the claims of third parties, whether or not Storj Labs International SEZC or Storj Inc has been advised of the possibility of such loss, however caused and on any theory of liability, arising out of or in connection with the possession, use or performance of this software.


Features:

 *imports the list of recipients to a local SQL
 * calculate the required amount of tokens based on the current price (using coinmarketcap API)
 * submits the transfers in batches and checks the status
 * export the results to a different csv

Supported transfer types:

 * Ethereum ERC-20 transfer
 * ZkSync (1.0) transfer
 * ZkSync (1.0) withdraw (source is on ZkSync, destination on Ethereum. Fees can be paid in ERC-20 tokens)
 * Polygon ERC-20 transfer (experimental)

## Requirements

* Ethereum node, fully synced (or an API provider)
* An ethereum account (spender) used to issue the ETH transactions.
* An ethereum account (owner) from which ERC-20 is transferred.
* **IMPORTANT**: since merging the zksync support application can be executed / compiled only
  with `zks-crypto-linux-x64.so`. see the instructions in the Zksync section below.

The spender and owner accounts can be the same account.

## Supported transfer flows

### Ethereum/Polygon ERC-20 `transfer()`

In this flow, the spender and owner are the same ETH account. The account must have both enough ETH to cover gas costs
and STORJ to cover the transfers.

This flow uses slightly less gas to issue the transactions but requires the account possess the STORJ.

### Ethereum/Polygon ERC-20 `transferFrom()`

In this flow, the spender and owner are different accounts. The spender needs ETH to cover the transactions and also  needs to be granted an allowance from the owner for enough STORJ to cover the transfers.

It uses a little more gas than the `transfer()` method but does not require STORJ to be transferred to the spender account.

### ZkSync transfer

This flow transfers tokens from ZkSync to ZkSync. The ERC-20 token should be supported by the ZkSync network

### ZkSync withdraw

This flow transfers from ZkSync to the L1 Ethereum chain with the `withdraw` method. It's more expensive than a transfer but the fees can be paid in any supported ERC-20 tokens.

## Install

```
$ go install storj.io/crypto-batch-payment/cmd/crybapy
```

# License

This repository is currently licensed with the [AGPLv3](https://www.gnu.org/licenses/agpl-3.0.en.html) license.

For code released under the AGPLv3, we request that contributors sign our
[Contributor License Agreement (CLA)](https://docs.google.com/forms/d/e/1FAIpQLSdVzD5W8rx-J_jLaPuG31nbOzS8yhNIIu4yHvzonji6NeZ4ig/viewform) so that we can relicense the  code under Apache v2, or other licenses in the future.


## Contributing

If you have suggestions / fixes / requests, please open an issue here or a topic on
[our community forum](https://forum.storj.io/)

## Basic usage

First, import the CSV file containing the payout data. The payout information will be stored in a database in the data
directory (defaults to `./data`).

```
$ ./crybapy import ./test/test-payouts.csv
```

The `NAME` of the payout will be the name of the CSV file without the extension.

To run the payout when the spender and owner are the same:

```
$ ./crybapy run <NAME> ./path/to/spender.key
```

To run the payouts when the spender and owner different:

```
$ ./crybapy run <NAME> <PATH TO SPENDER KEY> --owner <OWNER ADDRESS>
```

The key file must contain the hex-encoded ECDSA private key for the account.

By default, the program uses the `./geth.ipc` file to contact the node (which can be a symlink into the `geth.ipc` used  by your geth node). The address can be overriden by providing the `--node-address` flag:

```
$ ./crybapy run test spender.key --node-address /path/to/geth.ipc
```

The node address can also be a URL if your node is not running locally (supports
`http`,`https`,`ws`,`wss`schemes):

```
$ ./crybapy run test spender.key --node-address http://192.168.1.199
```

# For developers

## Testing ethereum based payment locally

First, you need an ethereum chain. You can use either a public testnetwork or a private.

One easy way to have a local testnet is using `elek/localgeth`:

```
git clone https://github.com/elek/localgeth.git
cd localgeth
docker-compose up -d
```

Create an account for yourself and get some balance:

```
./faucet.sh 0x5350Cd178a2dC46D0A020c698819bE008924278A
```

Save your private key:

```
echo bf5516fbad104e9869b21dc888779280c45d8c33610986907429ba1dec5408d5 > /tmp/key
```

Deploy the contract:

```
payouts  erc20 deploy --chain-id 1337 --node-address http://localhost:8545 /tmp/key

...
Deployed contract to address 0x3982BEB4e4370E5265B0232d18c0EE8Ca316a1BC
...
```

Write down your contract address!

```
export CONTRACT_ADDRESS=0x3982BEB4e4370E5265B0232d18c0EE8Ca316a1BC
```

Create a sample payment csv (eg. `payout.csv`):

```
addr,amnt
0xB766AE0dd39F26bC5E8f60fC1C95566eCFE3851f,10
```

Initialize the db:

```
payouts imports payouts.csv
```

Run the payout:

```
payouts run --max-gas 1000000000 \
  --gas-tip-cap 200 \
  --node-address=http://localhost:8545 \
  --contract $CONTRACT_ADDRESS \
  --quote-cache-expiry=60s \
  --chain-id 1337 payouts \
  /tmp/key
```

Audit the changes:

```
payouts audit --node-address=http://localhost:8545 --chain-id 1337 payouts.csv 

Auditing "payouts.csv"...
...
```

## ZkSync support

ZkSync transport

### Test on rinkeby test network

Download native library from https://github.com/zksync-sdk/zksync-crypto-c/releases/tag/v0.1.2

For example:

```bash
wget https://github.com/zksync-sdk/zksync-crypto-c/releases/download/v0.1.2/zks-crypto-linux-x64.so -O libzks-crypto.so
```

Set LD_LIBRARY_PATH

```bash
export LD_LIBRARY_PATH=`pwd`
export CGO_LDFLAGS=-L$LD_LIBRARY_PATH
```

Note: second one is required only if you do `go install/test/build`

Use an account which has STORJ balance on zksync rinkeby test network. (check it
here: https://rinkeby.zksync.io/account)

Note: you can mint STORJ tokens on rinkeby network with clicking to `Add fund` on the wallet page. After successful
minting the tokens can be deposited to zksync.

Save the private key to a file (eg. `key`)

Create a transaction list CSV and import it:

Example CSV:

```
addr,amnt
0xB766AE0dd39F26bC5E8f60fC1C95566eCFE3851f,0.0001
```

Execute the transactions:

```
payouts run \
 --type=zksync \
 --node-address=https://rinkeby-api.zksync.io/ \
 --quote-cache-expiry=60s \
 --chain-id 4 \
  payouts key
```

(Note: for production use `--chain-id=1` and `--node-address=https://api.zksync.io` )

Check if payment type is `zksync` and approve (`y`)

Check the transactions with audit command:

```
payouts audit --node-address=https://rinkeby-api.zksync.io/ --chain-id 4 --type=zksync payouts.csv
```

In case of any error with a transaction you can use detailed state with using REST API.

For example if audit shows and error:

```
TX state mismatch on hash "sync-tx:d09bd21df70c70dfabc5f8cb565a9d5d4bc59b63f1a4edc7460af872b6d7b10d" (db="pending", node="failed")
Checking payouts status...
Payout of 0.0001 to 0xB766AE0dd39F26bC5E8f60fC1C95566eCFE3851f on line 2 has no confirmed transactions (pending=1 dropped=0 failed=0)
```

You can check it with

```
http https://rinkeby-api.zksync.io/api/v0.1/transactions_all/0xd09bd21df70c70dfabc5f8cb565a9d5d4bc59b63f1a4edc7460af872b6d7b10d
```

(or using curl + jq).

Example output:

```
{
    "amount": "5868",
    "batch_id": null,
    "block_number": 72062,
    "created_at": "2021-12-16T11:32:44.985298",
    "fail_reason": "Not enough balance",
    ...
}
```

## ZkWithdraw support

There is a special case of payment on ZkSync chain: when we use `Withdraw` transactions instead of `Transfer`. It can
deliver payment on L1 (ethereum) but with paying fees in STORJ tokens.

To use this method you should use `zkwithdraw` type instead of `zksync`:

```
payouts run \
--type=zkwithdraw \
--node-address=https://rinkeby-api.zksync.io/ \
--quote-cache-expiry=60s \
payouts key
```

## Polygon support

As polygon is Ethereum compatible ([including EIP-1559 support]())

Official polygon bridge already created a token an ERC-20 contract on Polygon PoS which is bridged to STORJ ethereum.

The address of the contract is 0x355b8e02e7f5301e6fac9b7cac1d6d9c86c0343f

STORJ Polygon POS
contract: [0xd72357dAcA2cF11A5F155b9FF7880E595A3F5792](https://polygonscan.com/token/0xd72357dAcA2cF11A5F155b9FF7880E595A3F5792)

This is a proxy contract where the implementation (as of today)
is [0x3e1e043c84cb4f306dca24a71b411c29d4ed85f2](https://polygonscan.com/address/0x3e1e043c84cb4f306dca24a71b411c29d4ed85f2)

The proxy owner is 0x355b8e02e7f5301e6fac9b7cac1d6d9c86c0343f which
is [one of the Polygon Multisig keys](https://docs.polygon.technology/docs/faq/commit-chain-multisigs/) with the
following owners:

* Quickswap
* Curve
* Polygon
* Horizon Games
* Cometh

## Troubleshooting

Transactions can have 4 states (`pending`, `failed`, `canceled`, `confirmed`).

1. The application first creates a `pending` transaction and saves it the db.
2. Next it tries to submit to the chain
3. And it polls for the status until it's either `confirmed` or `failed`

Failure will stop the process but the process can be continued with re-executing the same command: the failure will be
ignored.

In case of any error, please check the `pending` lines in the db. For example:

```bash
 echo "select hash from tx where state = 'pending'" | sqlite3 payout/payouts.db
```

You can delete all these lines to trigger a new attempt, **but only if you are 100% sure that they are not on the
chain**, to avoid double payment.

Please check all the pending with eth/zksync api or explorer before deleting any pending item:

For example with ZkSync you can do:

```bash
echo "select hash from tx where state = 'pending'" | sqlite3 payout/payouts.db  | awk -F ':' '{print $2}' | xargs -IHASH curl https://api.zksync.io/api/v0.1/transactions_all/0xHASH        
```

And if all are really failed, you can delete them with:

```
echo "delete from tx where state='pending';" | sqlite3 payout/payouts.db
```
