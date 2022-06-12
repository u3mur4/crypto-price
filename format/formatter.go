package format

import "github.com/u3mur4/crypto-price/exchange"

// Formatter displays the markets to the user
type Formatter interface {
	Open()
	Show(market exchange.Market)
	Close()
}
