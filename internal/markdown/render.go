package markdown

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ---------------------------------------------------------------------------
// RenderResult — returned by RenderFull with extracted metadata
// ---------------------------------------------------------------------------

type RenderResult struct {
	HTML        string
	TOC         string   // pre-built <nav class="toc"> HTML
	Links       []string // internal link hrefs for backlinks
	Description string   // first paragraph plain text for OG
}

// ---------------------------------------------------------------------------
// mdLinkTransformer — strips .md extensions from relative link destinations
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// calloutTransformer — converts blockquotes starting with [!TYPE] to callouts
// ---------------------------------------------------------------------------

var calloutTypes = map[string]bool{
	"NOTE": true, "WARNING": true, "TIP": true,
	"IMPORTANT": true, "CAUTION": true,
}

var calloutRegex = regexp.MustCompile(`^\[!(NOTE|WARNING|TIP|IMPORTANT|CAUTION)\]\s*`)

type calloutTransformer struct{}

func (t *calloutTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		bq, ok := n.(*ast.Blockquote)
		if !ok {
			return ast.WalkContinue, nil
		}
		// First child should be a paragraph
		firstChild := bq.FirstChild()
		if firstChild == nil {
			return ast.WalkContinue, nil
		}
		para, ok := firstChild.(*ast.Paragraph)
		if !ok {
			return ast.WalkContinue, nil
		}
		// Get the text content of the first text segment
		firstText := para.FirstChild()
		if firstText == nil {
			return ast.WalkContinue, nil
		}
		textNode, ok := firstText.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}
		line := string(textNode.Segment.Value(source))
		match := calloutRegex.FindStringSubmatch(line)
		if match == nil {
			return ast.WalkContinue, nil
		}
		calloutType := strings.ToLower(match[1])
		bq.SetAttributeString("class", "callout callout-"+calloutType)

		// Strip the [!TYPE] prefix from the text
		remaining := strings.TrimSpace(line[len(match[0]):])
		if remaining == "" {
			// If the whole text node was just the marker, remove it
			// If there's a softline break after, remove that too
			next := textNode.NextSibling()
			para.RemoveChild(para, textNode)
			if next != nil {
				if _, isBr := next.(*ast.Text); isBr {
					seg := next.(*ast.Text).Segment
					val := string(seg.Value(source))
					if strings.TrimSpace(val) == "" {
						para.RemoveChild(para, next)
					}
				}
			}
		} else {
			// Replace segment with trimmed text by injecting a raw text node
			newSeg := text.NewSegment(textNode.Segment.Start+len(match[0]), textNode.Segment.Stop)
			textNode.Segment = newSeg
		}

		return ast.WalkContinue, nil
	})
}

// ---------------------------------------------------------------------------
// Wikilinks — [[page]] and [[page|display text]]
// ---------------------------------------------------------------------------

// wikilinkParser is an inline parser that converts [[target]] or [[target|text]]
// into standard ast.Link nodes.
type wikilinkParser struct{}

func (p *wikilinkParser) Trigger() []byte {
	return []byte{'['}
}

func (p *wikilinkParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, seg := block.PeekLine()
	if len(line) < 4 || line[0] != '[' || line[1] != '[' {
		return nil
	}

	// Find closing ]]
	end := bytes.Index(line[2:], []byte("]]"))
	if end < 0 {
		return nil
	}

	inner := string(line[2 : 2+end])
	if inner == "" {
		return nil
	}

	// Advance reader past [[ ... ]]
	block.Advance(2 + end + 2)

	target := inner
	display := inner
	if idx := strings.Index(inner, "|"); idx >= 0 {
		target = inner[:idx]
		display = inner[idx+1:]
	}

	// Normalize target: spaces → hyphens, lowercase
	target = strings.TrimSpace(target)
	target = strings.ReplaceAll(target, " ", "-")
	target = strings.ToLower(target)

	link := ast.NewLink()
	link.Destination = []byte("/" + target)
	_ = seg // suppress unused
	link.AppendChild(link, ast.NewString([]byte(display)))
	return link
}

type wikilinkExtension struct{}

func (e *wikilinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.PrioritizedValue{Value: &wikilinkParser{}, Priority: 99},
		),
	)
}

// ---------------------------------------------------------------------------
// calloutRendererExtension — renders blockquotes with class attributes
// ---------------------------------------------------------------------------

// We need a custom renderer for blockquotes that respects attributes
type calloutRenderer struct {
	html.Config
}

func newCalloutRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &calloutRenderer{Config: html.NewConfig()}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *calloutRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
}

func (r *calloutRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		class, ok := node.AttributeString("class")
		if ok {
			_, _ = w.WriteString(`<blockquote class="`)
			_, _ = w.Write(class.([]byte))
			_, _ = w.WriteString("\">\n")
		} else {
			_, _ = w.WriteString("<blockquote>\n")
		}
	} else {
		_, _ = w.WriteString("</blockquote>\n")
	}
	return ast.WalkContinue, nil
}

type calloutRendererExtension struct{}

func (e *calloutRendererExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.PrioritizedValue{Value: newCalloutRenderer(), Priority: 99},
		),
	)
}

// ---------------------------------------------------------------------------
// Goldmark instance
// ---------------------------------------------------------------------------

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.NewFootnote(),
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
			),
			&wikilinkExtension{},
			&calloutRendererExtension{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.PrioritizedValue{Value: &mdLinkTransformer{}, Priority: 100},
				util.PrioritizedValue{Value: &calloutTransformer{}, Priority: 99},
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

// Render converts markdown source to HTML.
func Render(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var titleRegex = regexp.MustCompile(`(?m)^#\s+(.+)$`)

// ExtractTitle extracts the first H1 heading from markdown source.
func ExtractTitle(source []byte) string {
	match := titleRegex.FindSubmatch(source)
	if match == nil {
		return ""
	}
	return string(bytes.TrimSpace(match[1]))
}

// ---------------------------------------------------------------------------
// RenderFull — renders markdown and extracts TOC, links, description
// ---------------------------------------------------------------------------

// tocEntry represents a heading for TOC generation.
type tocEntry struct {
	Level int
	ID    string
	Text  string
}

// RenderFull renders markdown and extracts metadata for TOC, backlinks, and OG.
func RenderFull(source []byte) (*RenderResult, error) {
	// Render HTML
	htmlBytes, err := Render(source)
	if err != nil {
		return nil, err
	}

	// Parse AST for extraction
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	result := &RenderResult{
		HTML: string(htmlBytes),
	}

	var headings []tocEntry
	var links []string
	var descFound bool

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			id := generateHeadingID(node, source)
			txt := collectText(node, source)
			headings = append(headings, tocEntry{
				Level: node.Level,
				ID:    id,
				Text:  txt,
			})

		case *ast.Link:
			dest := string(node.Destination)
			u, uerr := url.Parse(dest)
			if uerr == nil && u.Scheme == "" && !strings.HasPrefix(dest, "#") {
				// Relative link — internal
				links = append(links, dest)
			}

		case *ast.Paragraph:
			if !descFound && n.Parent() != nil && n.Parent().Kind() == ast.KindDocument {
				// Skip if inside a footnote
				if isInsideFootnote(n) {
					return ast.WalkContinue, nil
				}
				txt := collectText(node, source)
				if txt != "" {
					descFound = true
					if len(txt) > 160 {
						txt = txt[:160]
					}
					result.Description = txt
				}
			}
		}

		return ast.WalkContinue, nil
	})

	result.Links = links
	result.TOC = buildTOCHTML(headings)

	return result, nil
}

// isInsideFootnote checks if a node is inside a footnote definition.
func isInsideFootnote(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == east.KindFootnote {
			return true
		}
	}
	return false
}

// generateHeadingID generates the same ID that goldmark's auto heading ID would.
func generateHeadingID(heading *ast.Heading, source []byte) string {
	txt := collectText(heading, source)
	return slugify(txt)
}

// slugify converts text to a URL-friendly slug.
func slugify(s string) string {
	var buf strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			buf.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && buf.Len() > 0 {
				buf.WriteByte('-')
				prevDash = true
			}
		}
	}
	result := buf.String()
	return strings.TrimRight(result, "-")
}

// collectText gathers all text content from a node's children.
func collectText(n ast.Node, source []byte) string {
	var buf strings.Builder
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		} else if t, ok := child.(*ast.String); ok {
			buf.Write(t.Value)
		} else if _, ok := child.(*ast.CodeSpan); ok && entering {
			// Skip — text inside code spans is collected via Text nodes
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

// buildTOCHTML generates a <nav> element with a nested list of headings.
func buildTOCHTML(headings []tocEntry) string {
	// Filter to h2-h4 only
	var filtered []tocEntry
	for _, h := range headings {
		if h.Level >= 2 && h.Level <= 4 {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) < 2 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("<nav class=\"toc\">\n<ol>\n")
	for _, h := range filtered {
		class := fmt.Sprintf("toc-h%d", h.Level)
		buf.WriteString(fmt.Sprintf("  <li class=\"%s\"><a href=\"#%s\">%s</a></li>\n", class, h.ID, h.Text))
	}
	buf.WriteString("</ol>\n</nav>\n")
	return buf.String()
}
