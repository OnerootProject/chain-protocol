# DEX Protocol Spec

| KIP | title | author | discussions-to | status | type | created | requires | replaces |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| KIP22 | DEX Protocol | Charlie |  | Draft | Layer2 |  |  |  |

## Progress
Functions implemented
1. deposit
2. delegateWithdraw
3. trade
4. setBonusWhitelist
5. getBonuWhitelist
6. order canceled by user
7. balanceOf
8. orderState
9. management：setRelay,setAdmin and hard-coded owner
10.  statistics trade volume of block and bonus;
11. 2PC withdraw
12. support multi sig

**TODO List：**
1. batch method：trade，cancel
3. withdraw fee for super node



## Abstract

## WHAT

## WHY

## DETAIL

### Protocol Interface SPEC

| **Interface** | **Desc** | **Permission** | Progress |
| --- | --- | --- | --- |
| Modify Ope |  |  |  |
| deposit | deposit from wallet to DEX | All User | Done |
| withdraw | withdraw from dex to wallet | All User |  |
| 2PC withdraw | withdraw in 2-phase commit | All User | Done |
| delegateWithdraw | user sign the withdraw and submit by relay | relay | Done |
| trade | settle orders | relay | Done |
| list | list trade pair | admin | Done |
| unlist | unlist trade pair | admin | Done |
| setRelay | set relay | admin | Done |
| setAdmin | set admin | owner | Done |
| State Ope |  |  |  |
| balanceOf | balance of asset in dex | All User | Done |
| orderState | the order state | All User | Done |
| listed | get listed trade pair | All User | Done |
| isAdmin | check is admin | All User | Done |
| isRelay | check is relay | All User | Done |



#### deposit
User deposit token from wallet to DEX.If deposit success, block chain will emit `deposit` event.
Relay should listen this event to update their account balance.

| **Params** | **Type** | **Desc** |
| --- | --- | --- |
| asset | address | asset address/id |
| from | address | who deposit |
| to | address | the address receive the asset deposited |
| amount | uint64 | deposit amount |


#### delegateWithdraw
This method is called by relay only.The relay should help to withdraw after user has signed the withdraw message.
delegateWithdraw process:
1. user sign withdraw message ->
1. relay get the withdraw request ->
1. relay call `delegateWithdraw` of DEX contract ->
1. withdraw success

| **Params** | **Type** | **Desc** |
| --- | --- | --- |
| asset | address | asset address/id |
| from | address | withdraw address |
| to | address | receive address |
| amount | uint64 | amount |
| fee | uint64 | withdraw fee |
| salt | uint64 | nonce |
| extra | string | extra info,option |
| sig | Sig | signature from user |
|   Key | []byte | public key |
|   sig_data | []byte | signature data |
| relay | address | relay address |

to generate the signature：
> sig=SIGN(SHA256(asset|from|to|amount|fee|salt|extra),private_key)

#### 2PC Withdraw

2-phase commit is to let user withdraw asset from dex to wallet freely.Different from delegateWithdraw,
2PC withdraw don't need any other third-party,which is to make sure the user can freely control their own assets.
2-phase commit withdraw contains 2 steps:
1. call `prepareWithdraw` method to prepare withdraw first;
2. call `commitWithdraw` to withdraw asset at least `60 seconds` after prepare withdraw;

##### prepareWithdraw

| **Params** | **Type** | **Desc** |
| --- | --- | --- |
| asset | address | asset address/id |
| from | address | withdraw address |
| to | address | receive address |
| amount | uint64 | amount |

##### commitWithdraw
`commitWithdraw` will withdraw available asset from dex to wallet.
> Notice: prepared withdraw amount maybe greater than balance in dex,then the actual withdraw amount is balance in dex

after commit withdraw,the prepared withdraw will be reset to 0

| **Params** | **Type** | **Desc** |
| --- | --- | --- |
| asset | address | asset address/id |
| from | address | withdraw address |



#### trade

Trade method is responsible for matching two orders and settlement. After relay match the order successfully,
the trade method is called as the parameter of the matching order data, and the final chain settlement is performed.
For the trade method to be successful, a series of checks are required to ensure that no errors occur.
To do this, you need to do the following verification checks:

* The order is signed correctly;
* The order is the same trade pair;
* The order is not expired;
* Contains buy-sell orders;
* The order is not fully filled;
* The price of the maker and taker order matches;
* The volume submitted by relay is less than the smaller of the unfilled orders



| **Params** | **Type** | **Desc** |
| --- | --- | --- |
| MakerOrder | OrderData | maker order |
|   user | address | user |
|   pair | string | trade pair（asset address like：AAAA_BBBB） |
|   side | string | trade side:buy,sell |
|   price | string | 8 decimals at most.eg:0.00000001 |
|   amount | string | 8 decimals at most.eg:0.00000001  |
|   channel | string | channel address who help to collect the order.channel receive most of the fee |
|   fee | uint64 | fee rate,between 0 and 10000. |
|   fee | uint32 | expire time in unix seconds.0 is never expired |
|   salt | uint64 | nonce |
|   sig | Sig | signature |
| TakerOrder | OrderData | taker order |
| Relay | RelayArgs | relay params |
|   from | address | relay address |
|   tradeAmount | string | quoteToken amount |
|   makerFee | string | maker fee |
|   takerFee | string | taker fee |


##### Order data format

The user's order is signed by the their own private key off the chain, so it can represent the user's order to place.
In order to completely represent the order information when the user places a order, the order data needs to be signed includes:
* `user`: user account, base58 format
* `pair`: trade pair, but AAA_BBB, where AAA is the hex format of the asset address/id
* `side`: `buy` or `sell` direction.
* `price`: limit price, up to 8 decimal places, such as "0.12345670", more than 8 decimal places will be rejected;
* `amount`: The number of orders. The accuracy should be consistent with corresponding asset, otherwise will be rejected;
* `channel`: channel is the one who collect the orders for relay and will get most of the fee;
* `fee`: fee rate that user would like to pay.A number between 0 and 10000,trade fee=trade amount*fee/10000;
* `expire`: expire time of the order.0 means order never expired;
* salt: random number. Guarantee the uniqueness of the order ID;



To generate order signature：
> orderId=SHA256(user|pair|side|price|amount|channel|fee|expire|salt)
> sig=SIGN(orderId)



##### Order Matching

In order to ensure the consistent order on chain and off chain, Relay's order matching logic should be consistent with the DEX contract.
The protocol uses traditional Maker-Taker order matching logic as follows:
* buyer price>=seller price, the price match is successful;
* The trade price is the price of Maker;
* The trade amount is provided by relay, but cannot be greater than the smaller of unfilled order;


##### Fee Calculation
FeeRate is defined as `fee` in order data when user sign the order and sent to the channel address.
`fee` should between 0-10000,when order matched,trade fee will be calculated by `TradeAmount*fee/10000`.


### Protocol Upgrade

In order to support the continuous improvement of the protocol, the principles that need to be followed in the design for protocol upgrade are as follows:
1. In the case of the same data structure, the upgrade protocol should not affect the balance in the user's DEX account (the user is not required to transfer funds out of DEX when upgrading the agreement)
1. After the protocol is upgraded, the original signed order should not be valid in the new version of the protocol (need to add the protocol version version parameter to the order signature?)

## Copyright
Copyright and related rights waived via [CC0](https://creativecommons.org/publicdomain/zero/1.0/).

