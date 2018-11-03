# Crypto Price Tracker

![Price Tracker](price.gif)

```
$ crypto-price --help
Realtime Crypto Price Tracker

Usage:
  crypto-price [flags] {exchange:ticker}...

Flags:
  -h, --help              help for crypto-price
      --i3bar             i3bar format (default true)
      --json              json format
      --jsonl             json line format
      --satoshi           convert btc market price to satoshi
  -t, --template string   golang template format
      --version           version for crypto-price
```

example: `crypto-price --satoshi coinbase:btc-usd bittrex:btc-amp binance:xrp-btc`

i3blocks config:
```
[crypto]
command=/path/to/crypto-price --satoshi coinbase:btc-usd bittrex:btc-amp bittrex:btc-xrp
interval=persist
markup=pango
```

## Suported exchanges
[//]: # (badges stolen from https://github.com/ccxt/ccxt)
|                                                                                                                 | id       | name                                     |
|-----------------------------------------------------------------------------------------------------------------|----------|------------------------------------------|
|![binance](https://user-images.githubusercontent.com/1294454/29604020-d5483cdc-87ee-11e7-94c7-d1a8d9169293.jpg)  | binance  | [Binance](https://www.binance.com/)      |
|![coinbase](https://user-images.githubusercontent.com/1294454/41764625-63b7ffde-760a-11e8-996d-a6328fa9347a.jpg) | coinbase | [Coinbase Pro](https://pro.coinbase.com) |
|![bittrex](https://user-images.githubusercontent.com/1294454/27766352-cf0b3c26-5ed5-11e7-82b7-f3826b7a97d8.jpg)  | bittrex  | [Bittrex](https://bittrex.com)           |
|![cryptopia](https://user-images.githubusercontent.com/1294454/29484394-7b4ea6e2-84c6-11e7-83e5-1fccf4b2dc81.jpg)  | cryptopia  | [Cryptopia](https://www.cryptopia.co.nz)           |

## Supported Output formats
  - json
  - json line
  - i3bar
  - golang template

## Build
```bash
go get -d github.com/u3mur4/crypto-price
cd $GOPATH/src/github.com/u3mur4/crypto-price
dep ensure -vendor-only
go install
```

## Download binary (only for linux)
  - Check [releases](https://github.com/u3mur4/crypto-price/releases)
