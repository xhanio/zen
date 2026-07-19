package card

import "github.com/xhanio/errors"

const (
	formatMarkdown = "markdown"
	formatHTML     = "html"
	formatText     = "text"
)

func validateFormat(f string) error {
	switch f {
	case formatMarkdown, formatHTML, formatText:
		return nil
	}
	return errors.BadRequest.Newf("format must be one of markdown, html, text; got %q", f)
}
