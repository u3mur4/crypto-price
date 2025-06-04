# Crypto Price Tracker

![Price Tracker](price.gif)

Realtime Crypto Price Tracker for your status bar and terminal.

## Supported Exchanges

Currently, the primary supported exchange is:

*   **Binance:** (`binance`)

## Installation

```bash
go install github.com/u3mur4/crypto-price/cmd/crypto-price@latest
```

## Usage

The basic command structure is:

```bash
crypto-price [flags] {exchange:ticker}...
```

**Example:** Track BTC/USDT and ETH/BTC from Binance:

```bash
crypto-price binance:btc-usdt binance:eth-btc --waybar
```

### Command-line Flags

```
Usage:
  crypto-price [flags] {exchange:ticker}...

Flags:
      --alert                   Enable alerts. See "Configuring Alerts" section.
      --debug                   Enable debug log (default true). Logs to /tmp/crypto-tracker.log.
  -h, --help                    Help for crypto-price.
      --json                    Output in JSON format.
      --polybar                 Output in Polybar format.
      --polybar-weekend-short   Use short display on weekends for Polybar.
      --satoshi                 Convert BTC market prices to Satoshi.
      --server                  Start an HTTP server to expose market data.
  -t, --template string         Output in a custom format using Go templates.
      --waybar                  Output in Waybar format.
      --waybar-weekend-short    Use short display on weekends for Waybar.
```

**Market Format:** `{exchange:base-quote}`
*   `exchange`: The ID of the exchange (e.g., `binance`).
*   `base`: The base currency symbol (e.g., `btc`).
*   `quote`: The quote currency symbol (e.g., `usdt`).

Example: `binance:btc-usdt`

## Observers (Output Formats)

`crypto-price` can output data in several formats, suitable for different use cases.

### JSON Output (`--json`)

Outputs market data in JSON format, one object per update for each tracked market.

**Command:**
```bash
crypto-price binance:btc-usdt --json
```

**Example Output:**
```json
{"exchange":"binance","base":"btc","quote":"usdt","candle":{"high":106000,"open":105376.9,"close":105708.29,"low":105132.27,"percent":0.3144806878927042,"color":"#f0f6f0"}}
```
*   `color`: Hex color code representing the price change (green for up, red for down, white for neutral).
*   `percent`: Percentage change from the opening price of the 1-day candle.

### Polybar Output (`--polybar`)

Formats output for Polybar, including color based on price change and click actions.

**Command:**
```bash
crypto-price binance:btc-usdt --polybar
```

**Example Output (for BTC/USDT):**
```
%{F#f0f6f0}%{A1:curl -d 'action=toggle_price&market=binance:btc-usdt' -X POST http://localhost:60253/polybar:}BTC%{A}: $105708 (+0.3%) %{F-}
```
*   The color (e.g., `#f0f6f0`) changes based on price movement.
*   Clicking on the currency symbol (e.g., `BTC`) toggles the visibility of the price and percentage change. This action sends a POST request to `http://localhost:60253/polybar`.

**Polybar Module Configuration:**
```ini
[module/crypto]
type = custom/script
exec = crypto-price binance:btc-usdt --polybar
tail = true
```

*   The `--polybar-weekend-short` flag can be added to `exec` to show only the symbol on weekends.
*   The click action requires `curl` to be installed.
*   The internal HTTP server for Polybar interactions runs on port `60253`.

### Waybar Output (`--waybar`)

Formats output for Waybar, using Pango markup for colors and supporting click actions.

**Command:**
```bash
crypto-price binance:btc-usdt --waybar
```

**Example Output (for BTC/USDT):**
```html
<span color='#f0f6f0'>BTC: $105708.290 (+0.3%) </span>
```
*   The `color` attribute changes based on price movement.

**Waybar Module Configuration:**
```json
"custom/crypto-price": {
  "exec": "crypto-price binance:btc-usdt --waybar --waybar-weekend-short",
  "on-click": "curl -d 'action=toggle_price&market=binance:btc-usdt' -X POST http://localhost:60254/waybar",
  "on-click-right": "curl -d 'acton=toggle_color&market=binance:btc-usdt' -X POST http://localhost:60254/waybar",
}
```
*   Add `--waybar-weekend-short` to `exec` to show only the symbol on weekends.
*   `on-click`: Toggles price visibility for the specified market.
*   `on-click-right`: Toggles color highlighting for the specified market.
*   These click actions require `curl`.
*   The internal HTTP server for Waybar interactions runs on port `60254`. Ensure the `market` value in the `curl` command (e.g., `binance:btc-usdt`) matches one of the markets `crypto-price` is tracking.

### Go Template Output (`-t, --template`)

Allows for custom output formatting using Go's text/template package.

**Command:**
```bash
crypto-price binance:btc-usdt -t "{{.Base}}/{{.Quote}}: {{.Candle.Close}} ({{printf \"%.2f\" .Candle.Percent}}%) {{.LastUpdate.Format \"15:04:05\"}}{{printf \"\n\"}}"
```

**Example Output (for the template above):**
```
BTC/USDT: 105708.29 (0.31%) 10:30:45
```

**Available Template Data:**
The template is executed with a `Market` struct:
```go
type Market struct {
    Exchange   string    // e.g., "binance"
    Base       string    // e.g., "btc"
    Quote      string    // e.g., "usdt"
    Candle     Candle    // See below
    LastUpdate time.Time // Time of the last price update
}

type Candle struct {
    Open  float64 // Opening price of the 1-day candle
    High  float64 // Highest price of the 1-day candle
    Low   float64 // Lowest price of the 1-day candle
    Close float64 // Current closing price
}

// Candle also has a .Percent() method:
// .Candle.Percent   // Returns float64, e.g., 0.3144806878927042
```

### Server Mode (`--server`)

Starts an HTTP server on port `23232` that exposes market data via a JSON API.

**Command:**
```bash
crypto-price binance:btc-usdt --server
```

**API Endpoint:**
`http://localhost:23232/api/{exchange}:{base}-{quote}`

**Example Request:**
```bash
curl http://localhost:23232/api/binance:btc-usdt
```

**Example Response (similar to JSON output):**
```json
{"exchange":"binance","base":"btc","quote":"usdt","candle":{"high":106000,"open":105376.9,"close":105708.29,"low":105132.27,"percent":0.3144806878927042,"color":"#f0f6f0"}}
```

## Configuring Alerts (`--alert`)

When the `--alert` flag is used, `crypto-price` will monitor markets and trigger custom commands based on defined conditions. Alerts are configured in a JSON file located at:
`~/.config/crypto-alerts/alerts.json` (on Linux) or the equivalent user config directory on other OSes.

The application watches this file for changes and reloads alerts automatically.

**`alerts.json` Structure:**
The file should contain an array of alert definitions.
```json
[
  {
    "id": "binance:btc-usdt",
    "enabled": true,
    "grace_period": "15m",
    "condition": "gt_price",
    "value": [60000],
    "cmd": "notify-send 'BTC Alert!' 'BTC is over $60000'"
  },
]
```

**Alert Definition Fields:**

*   `id` (string, required): The market identifier, e.g., `binance:btc-usdt`.
*   `enabled` (boolean, required): Set to `true` to enable the alert, `false` to disable.
*   `grace_period` (string, optional): Duration to wait before this alert can be triggered again (e.g., "5m", "1h30m"). If not set or invalid, alert can trigger on every matching update.
*   `last_alert` (timestamp, internal): Not meant to be manually set in the config. Used by the application.
*   `condition` (string, required): The condition to check.
    *   `gt_percent`: Triggers if `Candle.Percent()` is greater than `value[0]`.
    *   `lt_percent`: Triggers if `Candle.Percent()` is less than `value[0]`.
    *   `gt_price`: Triggers if `Candle.Close` (current price) is greater than `value[0]`.
    *   `lt_price`: Triggers if `Candle.Close` (current price) is less than `value[0]`.
*   `value` (array of float, required): The threshold value(s) for the condition. Currently, only the first element `value[0]` is used.
*   `cmd` (string, required): The command to execute when the alert triggers. The command is parsed using shellwords.

## License

This project is licensed under the terms of the [LICENSE](LICENSE) file.
