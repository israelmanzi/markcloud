package markdown

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mdLinkTransformer strips .md extensions from relative link destinations.
type mdLinkTransformer struct{}

func (t *mdLinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		dest := string(link.Destination)
		u, err := url.Parse(dest)
		if err != nil || u.Scheme != "" {
			return ast.WalkContinue, nil
		}
		if strings.HasSuffix(u.Path, ".md") {
			u.Path = strings.TrimSuffix(u.Path, ".md")
			link.Destination = []byte(u.String())
		}
		return ast.WalkContinue, nil
	})
}

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
			),
		),
		goldmark.WithParserOptions(
			parser.WithASTTransformers(
				util.PrioritizedValue{Value: &mdLinkTransformer{}, Priority: 100},
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
