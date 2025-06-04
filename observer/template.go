package observer

import (
	"io"
	"os"

	"github.com/alecthomas/template"
	"github.com/u3mur4/crypto-price/exchange"
)

type TemplateOutput struct {
	Output   io.Writer
	Template *template.Template
}

func NewTemplateOutput(format string) TemplateOutput {
	return TemplateOutput{
		Output:   os.Stdout,
		Template: template.Must(template.New("TemplateFormatter").Parse(format)),
	}
}

func (t TemplateOutput) Show(info exchange.MarketDisplayInfo) {
	t.Template.Execute(t.Output, info.Market)
}
