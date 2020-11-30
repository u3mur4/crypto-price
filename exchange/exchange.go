package exchange

import (
	"context"
	"time"
)

// Exchange listens for price changes in realtime
type Exchange interface {
	// Name of the exchange
	Name() string
	// Register a market to listen for price changes
	Register(base string, quote string, interval time.Duration) error
	// Start listening for price changes in the registered markets
	Start(ctx context.Context, update chan<- Chart) error
}

type exchangeHelper struct {
	name   string
	charts []*Chart
}

func (h *exchangeHelper) Name() string {
	return h.name
}

func (h *exchangeHelper) Register(base string, quote string, interval time.Duration) error {
	h.charts = append(h.charts, newChart(h.name, base, quote, interval))
	return nil
}
