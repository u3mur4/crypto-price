package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

func httpGetJSON(url string, v interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return err
	}
	return nil
}

func runEvery(ctx context.Context, d time.Duration, f func()) {
	go func() {
		for {
			goalTime := time.Now().Truncate(d).Add(d)
			wait := goalTime.Sub(time.Now())
			select {
			case <-time.After(wait):
				f()
			case <-ctx.Done():
				return
			}
		}
	}()
}
