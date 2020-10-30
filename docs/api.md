# Ortelius API

## X Chain API

| Name                                                    | Route
|---                                                      | ---
| [Search](#search---xsearch)                             | `/x/search`
| [Aggregates](#aggregates---xaggregates)                 | `/x/aggregates`
| [List Transactions](#list-transactions---xtransactions) | `/x/transactions`
| [Get Transaction](#get-transaction---xtransactionsid)   | `/x/transactions/:id`
| [List Assets](#list-assets---xassets)                   | `/x/assets`
| [Get Asset](#get-asset---xassetsalias_or_id)            | `/x/assets/:alias_or_id`
| [List Addresses](#list-addresses---xaddresses)          | `/x/addresses`
| [Get Address](#get-address---xaddressesid)              | `/x/addresses/:id`

### Search - `/x/search`

Searches for a Transaction, Asset, or Address based on its ID.

Params:

`query` (Required) - The ID to search for.

#### Response:

```json
{
  "count": 1,
  "results": [
    {
      "type": "transaction",
      "data": {}
    }
  ]
}
```

### List Transactions - `/x/transactions`

[*Global list params*](#Global-List-Params)

- `id` - A transaction ID to filter by.

- `chainID` - A chain ID to filter by. Maybe be passed multiple times. Supports X-chain and P-chain chains.

- `assetID` - A asset ID to filter by.

- `address` - An address ID to filter by. Maybe be passed multiple times.

- `startTime` - The Time to start calculating from. Defaults to the Time of the first known transaction. Valid values are unix timestamps (in seconds) or RFC3339 datetime strings.

- `endTime` - The Time to end calculating to. Defaults to the current Time. Valid values are unix timestamps (in seconds) or RFC3339 datetime strings.

- `disableGenesis` - Bool value = true will suppress genesis input/output records in results.

- `sort` - The sorting method to use. Options: `timestamp-asc`, `timestamp-desc`. Default: `timestamp-asc`


#### Response:

Array of transaction objects

### Get Transaction - `/x/transactions/:id`

#### Params:

- `id` - The ID of the transaction to get

#### Response

The transaction object

### Aggregate Transactions - `/x/aggregates`

(Previously was `/x/transactions/aggregates` which now aliases to this endpoint)

#### Params:

- `chainID` - The chain ID to aggregate. Maybe be passed multiple times. Supports X-chain and P-chain chains.

- `assetID` - The asset ID to aggregate.

- `startTime` - The Time to start calculating from. Defaults to the Time of the first known transaction. Valid values are unix timestamps (in seconds) or RFC3339 datetime strings.

- `endTime` - The Time to end calculating to. Defaults to the current Time. Valid values are unix timestamps (in seconds) or RFC3339 datetime strings.

- `intervalSize` - If given, a list of intervals of the given size from startTime to endTime will be returned, with the aggregates for each interval. Valid values are `minute`, `hour`, `day`, `week`, `month`, `year`, or a valid Go duration string as described here: https://golang.org/pkg/Time/#ParseDuration

- `version` - If 1 will use new asset aggregate tables

#### Response:

```json
{
    "startTime": "2019-11-01T00:00:00Z",
    "endTime": "2020-04-23T20:28:21.358567Z",
    "aggregates": {
        "transactionCount": 23,
        "transactionVolume": 719999999992757400,
        "outputCount": 1,
        "addressCount": 16,
        "assetCount": 3
    },
    "intervalSize": 2592000000000000,
    "intervals": [
        {
            "transactionCount": 22,
            "transactionVolume": 719999999992757399,
            "outputCount": 0,
            "addressCount": 15,
            "assetCount": 4
        },
        {
            "transactionCount": 1,
            "transactionVolume": 1,
            "outputCount": 0,
            "addressCount": 1,
            "assetCount": 1
        }
    ]
}
```

### List Assets - `/x/assets`

[*Global list params*](#Global-List-Params)

- `id` - An asset ID to filter by.

- `alias` - An alias string to filter by.

#### Response:

Array of asset objects

### Get Asset - `/x/assets/:alias_or_id`

#### Params:

- `alias_or_id` - The alias or ID of the asset to get

#### Response:

Array of asset objects

### List Addresses - `/x/addresses`

#### Params:

[*Global list params*](#Global-List-Params)

- `address` - An address ID to filter by.

- `version` - If 1 will use new asset aggregate count tables

#### Response:

Array of Address objects

```json
[
  "count": 1,
  {
    "address": "2poot6VNEurx99o5WZigk2ic3ssj2T5Fz",
    "publicKey": null,
    "transactionCount": 2,
    "balance": 0,
    "lifetimeValue": 26000,
    "utxoCount": 0
  },
  {
    "address": "6cesTteH62Y5mLoDBUASaBvCXuL2AthL",
    "publicKey": null,
    "transactionCount": 186,
    "balance": 0,
    "lifetimeValue": 8369999998480180000,
    "utxoCount": 0
  }
]
```

### Get Address - `/x/addresses/:address`

#### Params:

- `address` - The base58-encoded Address to show.

#### Response:

```json
{
  "address": "6Y3kysjF9jnHnYkdS9yGAuoHyae2eNmeV",
  "publicKey": null,
  "transactionCount": 1,
  "balance": 0,
  "lifetimeValue": 45000000000000000,
  "utxoCount": 0
}
```

## P-Chain API


## Global List Params

These parameters are applied to all List endpoints.

- `limit` -  The maximum number of items to return.
    - Max: `500`
    - Default: `500`

- `offset` - The number of items to skip.

- `disableCount` - Bool. Suppress counting in results, `count` will be 0 in response. If you are not using the returned counts you should set this to enable better performance.
