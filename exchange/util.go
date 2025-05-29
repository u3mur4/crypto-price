package exchange

import (
	"context"
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"time"
)

var httpClient = http.Client{Timeout: 5 * time.Second}

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

func HasInternetConnection() bool {
	// List of reliable IP:port targets (TCP)
	targets := []string{
		"8.8.8.8:53",        // Google DNS
		"1.1.1.1:53",        // Cloudflare DNS
		"9.9.9.9:53",        // Quad9 DNS
		"208.67.222.222:53", // OpenDNS
		"8.8.4.4:53",        // Google secondary
	}

	rand.Seed(time.Now().UnixNano())
	target := targets[rand.Intn(len(targets))]

	timeout := 500 * time.Millisecond
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
