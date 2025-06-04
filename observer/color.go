package observer

import (
	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/u3mur4/crypto-price/exchange"
)

// ColorMap contains that at specifix position what color should be used
// It is interpolating colors between positions
type colorMap []struct {
	Col colorful.Color
	Pos float64
}

// getInterpolatedColorFor calculates the interpolated color for t
func (gt colorMap) getInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(gt)-1; i++ {
		c1 := gt[i]
		c2 := gt[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendLuv(c2.Col, t).Clamped()
		}
	}

	return gt[len(gt)-1].Col
}

func mustParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}

var defaultColorMap = colorMap{
	{mustParseHex("#8b0000"), -100.0},
	{mustParseHex("#ff0000"), -7.0},
	{mustParseHex("#ffa500"), -5.0},
	{mustParseHex("#f8de00"), -3.0},
	{mustParseHex("#f5f5f5"), +0.0},
	{mustParseHex("#c4ffc4"), +3.0},
	{mustParseHex("#00d800"), +7.0},
	{mustParseHex("#009700"), +10.0},
	{mustParseHex("#007600"), +100.0},
}

func getInterpolatedColorFor(candle exchange.Candle) colorful.Color {
	return defaultColorMap.getInterpolatedColorFor(candle.Percent())
}
