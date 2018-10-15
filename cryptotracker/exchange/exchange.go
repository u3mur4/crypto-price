package exchange

// Exchange updates the price realtime
type Exchange interface {
	// Start listening price changes for the specified markets
	Listen(markets []string) (<-chan Market, <-chan error)
}
