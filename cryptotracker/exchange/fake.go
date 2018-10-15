package exchange

import (
	"time"
)

type fake struct {
	m Market
}

func (f fake) Listen(products []string) (<-chan Market, <-chan error) {
	updateChan := make(chan Market)
	errorChan := make(chan error)
	go f.listen(updateChan, errorChan)
	return updateChan, errorChan
}

func (f *fake) listen(updateChan chan<- Market, errorChan chan<- error) {
	defer close(errorChan)
	defer close(updateChan)

	for {
		time.Sleep(time.Millisecond * 100)
		f.m.ActualPrice += 1
		if f.m.ActualPrice/f.m.OpenPrice >= 2 {
			f.m.ActualPrice = 100
		}
		updateChan <- f.m
	}
}

func NewFake() Exchange {
	return &fake{
		m: Market{
			BaseCurrency:  "fake",
			QuoteCurrency: "mock",
			OpenPrice:     1000,
			ActualPrice:   800,
		},
	}
}
