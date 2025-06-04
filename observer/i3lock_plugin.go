package observer

import (
	"bytes"
	"context"
	"fmt"
	"image"
	imgcolor "image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/u3mur4/crypto-price/exchange"
)

var htmlTemplate = `
<body style="background-color: transparent;">
<!-- TradingView Widget BEGIN -->
<div class="tradingview-widget-container">
  <div class="tradingview-widget-container__widget"></div>
  <div class="tradingview-widget-copyright"><a href="https://www.tradingview.com/symbols/EURUSD/?exchange=FX" rel="noopener" target="_blank"><span class="blue-text">EUR USD rates</span></a> by TradingView</div>
  <script type="text/javascript" src="https://s3.tradingview.com/external-embedding/embed-widget-mini-symbol-overview.js" async>
  {
  "symbol": "%s",
  "width": 350,
  "height": 220,
  "locale": "en",
  "dateRange": "1D",
  "colorTheme": "light",
  "trendLineColor": "rgba(41, 98, 255, 1)",
  "underLineColor": "rgba(41, 98, 255, 0)",
  "underLineBottomColor": "rgba(41, 98, 255, 0)",
  "isTransparent": true,
  "autosize": false,
  "largeChartUrl": "",
  "noTimeScale": false,
  "chartOnly": false
}
  </script>
</div>
</body>
<!-- TradingView Widget END -->
`

type i3lockPlugin struct {
	X      int
	Y      int
	ctx    context.Context
	cancel context.CancelFunc
	market *exchange.Market
	// NOTE: I need to reload the page sometimes because the chart doesn't reflects the price change in tradingview
	lastReload time.Time
}

func newi3lockPlugin(X, Y int, market exchange.Market) *i3lockPlugin {
	// create context
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		// chromedp.WithDebugf(log.Printf),
	)

	// generate tradingview plugin html
	path := fmt.Sprintf("/tmp/tw-%s:%s%s.html", market.Exchange, market.Base, market.Quote)
	html := fmt.Sprintf(htmlTemplate, fmt.Sprintf("%s:%s%s", market.Exchange, market.Base, market.Quote))
	os.WriteFile(path, []byte(html), 0777)

	// load html and set transparency
	tasks := chromedp.Tasks{
		chromedp.Navigate("file://" + path),
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := emulation.SetDefaultBackgroundColorOverride().WithColor(
				&cdp.RGBA{
					R: 255,
					G: 255,
					B: 255,
					A: 0,
				}).Do(ctx)

			if err == nil {
				return nil
			}
			return fmt.Errorf("hide default white background: %w", err)
		}),

		chromedp.Sleep(time.Second * 2),
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		log.Fatal(err)
	}

	return &i3lockPlugin{
		X:          X,
		Y:          Y,
		ctx:        ctx,
		cancel:     cancel,
		market:     nil,
		lastReload: time.Now(),
	}
}

func (p *i3lockPlugin) takeScreenShot() ([]byte, error) {
	// capture screenshot of an element
	var imgData []byte
	tasks := chromedp.Tasks{}
	if time.Since(p.lastReload) > time.Minute*5 {
		tasks = append(tasks, chromedp.Reload(), chromedp.Sleep(time.Second * 2))
		p.lastReload = time.Now()
	}
	tasks = append(tasks, chromedp.Screenshot(".tradingview-widget-container > iframe", &imgData, chromedp.NodeVisible))
	if err := chromedp.Run(p.ctx, tasks); err != nil {
		return nil, err
	}
	return imgData, nil
}

func (p *i3lockPlugin) Update(info exchange.MarketDisplayInfo) (bool, error) {
	market := info.Market

	if !p.shouldCreateNewScreenshot(market) {
		return false, nil
	}
	p.market = &market
	imgDate, err := p.takeScreenShot()
	if err != nil {
		return false, err
	}
	err = p.fixAndSaveImg(imgDate)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *i3lockPlugin) shouldCreateNewScreenshot(market exchange.Market) bool {
	if p.market == nil {
		return true
	}

	withinPercent := func(n, target, percent float64) bool {
		// Calculate the maximum and minimum values that n can be within
		// the given percentage of target
		min := target * (1.0 - percent/100.0)
		max := target * (1.0 + percent/100.0)

		// Return whether n is within the range [min, max]
		return n >= min && n <= max
	}

	return !withinPercent(market.Candle.Close, p.market.Candle.Close, 0.1)
}

func (p *i3lockPlugin) fixAndSaveImg(imgData []byte) error {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return err
	}

	// remove tradingview logo
	newImg := image.NewRGBA(img.Bounds())
	draw.Draw(newImg, img.Bounds(), img, image.Point{}, draw.Src)
	transparent := imgcolor.Transparent
	draw.Draw(newImg, image.Rect(310, 0, 350, 40), &image.Uniform{transparent}, image.Point{}, draw.Src)

	// save img to i3lock plugin directory with proper filename format
	path := fmt.Sprintf("/tmp/i3lock/crypto-price.%s-pos:%d-%d.png", p.market.Base+p.market.Quote, p.X, p.Y)
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return png.Encode(f, newImg)
}

type i3lockUpdateRequest struct {
	BatchTime     time.Duration
	MaxWaitTime   time.Duration
	request       chan struct{}
	batchTicker   *time.Ticker
	maxWaitTicker *time.Ticker
}

func Newi3lockUpdateRequest() *i3lockUpdateRequest {
	r := &i3lockUpdateRequest{
		BatchTime:   time.Second * 2,
		MaxWaitTime: time.Second * 5,
		request:     make(chan struct{}, 1),
		batchTicker: time.NewTicker(time.Hour),
	}
	r.batchTicker.Stop()
	r.maxWaitTicker = time.NewTicker(r.MaxWaitTime)
	go r.listen()
	return r
}

func (i *i3lockUpdateRequest) listen() {
	update := false
	for {
		select {
		case <-i.request:
			i.batchTicker.Reset(i.BatchTime)
			update = true
			continue
		case <-i.batchTicker.C:
		case <-i.maxWaitTicker.C:
		}

		if update {
			update = false
			i.batchTicker.Stop()
			i.sendSignalToProcess("i3lock")
		}
	}
}

func (i *i3lockUpdateRequest) RequestUpdate() {
	i.request <- struct{}{}
}

func (i *i3lockUpdateRequest) sendSignalToProcess(name string) error {
	// Find the PID of the process with the given name
	pids, err := i.findPIDByName(name)
	if err != nil {
		return err
	}

	// Send the SIGUSR1 signal to the process
	for _, pid := range pids {
		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		err = process.Signal(syscall.SIGUSR1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *i3lockUpdateRequest) findPIDByName(name string) (pids []int, err error) {
	// Run the pidof command to find the PID of the process with the given name
	cmd := exec.Command("pidof", name)
	output, err := cmd.Output()
	if err != nil {
		return pids, fmt.Errorf("error running pidof: %v", err)
	}

	// Parse the output of the command to find the PID
	nums := strings.Split(string(output), " ")
	for _, num := range nums {
		pid, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			// fmt.Println(fmt.Errorf("error parsing PID: %v", err))
			continue
		}
		pids = append(pids, pid)
	}

	return pids, nil
}

type I3LockPluginFormatter struct {
	data      map[string]*i3lockPlugin
	updater   *i3lockUpdateRequest
	startYPos int
	pluginId  int
}

func NewI3LockPluginFormatter(Y int) *I3LockPluginFormatter {
	return &I3LockPluginFormatter{
		data:      make(map[string]*i3lockPlugin),
		updater:   Newi3lockUpdateRequest(),
		startYPos: Y,
		pluginId:  0,
	}
}

func (p *I3LockPluginFormatter) Open() {
	p.deleteFiles()
}

func (p *I3LockPluginFormatter) deleteFiles() error {
	pngFiles, err := filepath.Glob("/tmp/i3lock/crypto-price*.png")
	if err != nil {
		return err
	}

	// Iterate through the slice of png files and delete them
	for _, pngFile := range pngFiles {
		os.Remove(pngFile)
	}
	return nil
}

func (p *I3LockPluginFormatter) Show(info exchange.MarketDisplayInfo) {
	market := info.Market

	key := market.Exchange + market.Base + market.Quote

	var plugin *i3lockPlugin = nil
	if cacheData, ok := p.data[key]; !ok {
		plugin = newi3lockPlugin(50, p.startYPos+p.pluginId*200, market)
		p.pluginId += 1
		p.data[key] = plugin
	} else {
		plugin = cacheData
	}

	requestUpdate, err := plugin.Update(info)
	if err != nil {
		log.Println("i3lock plugin error: ", err)
		return
	}

	if requestUpdate {
		p.updater.RequestUpdate()
	}

}

func (p *I3LockPluginFormatter) Close() {
	p.deleteFiles()
}
