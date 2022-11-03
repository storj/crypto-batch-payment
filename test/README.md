# Integration test suite

This directory contains test scripts and test data to execute crypthopper.

Some scripts depend on real testnet, where real balance is required. Therefore, they couldn't be executed automatically.

## ZkSync (Rinkeby)

Rinkeby STORJ tokens (on L1 chain) can be claimed from the ZkSync (Rinkeby) wallet: https://rinkeby.zksync.io/account

Click to "Add Funds" and "Mint Tokens+"

Note: it gives you STORJ tokens on L1 which should be moved to the L2 before executing the test. Initial registration
also can be paid in STORJ tokens.

## Ethereum (Rinkeby)

It can be tested with custom ERC-20 token, but the easiest way is to use the Zksync Test STORJ (L1) contract which is 
connected to the faucet.

Contract address is: 0x8098165d982765097E4aa17138816e5b95f9fDb5

https://rinkeby.zksync.io/account

To cover transaction cost, ETH is required. Rinkeby faucet works only with twitter (if at all):

https://faucet.rinkeby.io/

Ethereum test requires ETH RPC backend. (AFAIK there is no available endpoint without registration).

Infura gives API access after registration:

```
export CRYPTHOPPER_NODE_ADDRESS=https://rinkeby.infura.io/v3/<YOUR_PROJECT_ID>
```

## Polygon (Mumbai)

Polygon faucet can be found here https://faucet.polygon.technology/.

 * Request MATIC to cover gas gee
 * Request PoS ERC-20 token

Contract address (on polygon) is 0xfe4f5145f6e09952a5ba9e956ed0c25e3fa4c7f1

Faucet gives 2 token, therefore the amounts are 

Scanner is at: https://mumbai.polygonscan.com/tx/