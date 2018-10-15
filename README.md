# Crypto Price Tracker

![Price Tracker](price.gif)

Realtime pricetracking.

usage: `crypto-price exchange:marketname`

example: `crypto-price --satoshi coinbase:btc-usd bittrex:btc-amp binance:xrp-btc`

The `--satoshi` flag converts btc market values to satoshi.

## Suported exchanges
  - bittrex
  - binance
  - coinbase

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
  
## Download (only for linux)
  - Check [releases](https://github.com/u3mur4/crypto-price/releases)
