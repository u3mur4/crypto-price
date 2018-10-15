# Crypto Price Tracker

![Price Tracker](price.gif)

```
$ crypto-price --help
Real Time Crypto Price Tracker

Usage:
  crypto-price [flags] {exchange:ticker}... 

Flags:
  -h, --help              help for crypto-price
      --i3bar             i3bar format (default true)
      --json              JSON format
      --jsonl             JSON line format
      --satoshi           Convert btc market price to satoshi
  -t, --template string   golang template format

```

example: `crypto-price --satoshi coinbase:btc-usd bittrex:btc-amp binance:xrp-btc`

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
