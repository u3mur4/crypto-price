package format

import (
	"io"
	"os"

	"github.com/alecthomas/template"
)

type templateFormat struct {
	Output   io.Writer
	Template *template.Template
}

// NewTemplate displays the market as the specified golang template format
func NewTemplate(format string) Formatter {
	return templateFormat{
		Output:   os.Stdout,
		Template: template.Must(template.New("TemplateFormatter").Parse(format)),
	}
}

func (t templateFormat) Open() {}

func (t templateFormat) Show(m Market) {
	t.Template.Execute(t.Output, m)
}

func (t templateFormat) Close() {}
