package exchange

import (
	"context"
)

// Exchange listens for price changes in realtime
type Exchange interface {
	// Register a market to listen for price changes
	Register(base string, quote string) error
	// Start listening for price changes in the registered markets
	Start(ctx context.Context, update chan<- Market) error
}
