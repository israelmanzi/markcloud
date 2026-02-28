package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
			),
		),
	)
}

func Render(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var titleRegex = regexp.MustCompile(`(?m)^#\s+(.+)$`)

func ExtractTitle(source []byte) string {
	match := titleRegex.FindSubmatch(source)
	if match == nil {
		return ""
	}
	return string(bytes.TrimSpace(match[1]))
}
