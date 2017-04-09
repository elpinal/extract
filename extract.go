package extract

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var tagNamesToIgnore = map[string]int{
	"aside":    0,
	"embed":    0,
	"form":     0,
	"head":     0,
	"iframe":   0,
	"link":     0,
	"meta":     0,
	"noscript": 0,
	"object":   0,
	"option":   0,
	"script":   0,
	"style":    0,
}

var tagNamesToIgnoreOnlyItself = map[string]int{
	"body": 0,
	"html": 0,
}

var negativePattern = []string{
	"breadcrumb",
	"combx",
	"comment",
	"contact",
	"disqus",
	"foot",
	"footer",
	"footnote",
	"header",
	"hidden",
	"link",
	"media",
	"meta",
	"mod-conversations",
	"pager",
	"pagination",
	"promo",
	"reaction",
	"related",
	"scroll",
	"share",
	"shoutbox",
	"sidebar",
	"social",
	"sponsor",
	"tags",
	"toolbox",
	"widget",
}

var positivePattern = []string{
	"article",
	"body",
	"content",
	"entry",
	"hentry",
	"page",
	"post",
	"text",
}

func isAlphabetic(s byte) bool {
	return ('A' <= s && s <= 'Z') || ('a' <= s && s <= 'z')
}

func indexWord(s, sep string) int {
	i := strings.Index(s, sep)
	n := len(sep)
	switch {
	case i < 0:
		return i
	case len(s) == n:
		return i
	case i == 0 && isAlphabetic(s[n]):
		return -1
	case i > 0 && isAlphabetic(s[i-1]):
		return -1
	}
	return i
}

func removeChild(n, c *html.Node) {
	if c.Parent != n {
		panic("html: removeChild called for a non-child Node")
	}
	if n.FirstChild == c {
		n.FirstChild = c.NextSibling
	}
	if c.NextSibling != nil {
		c.NextSibling.PrevSibling = c.PrevSibling
	}
	if n.LastChild == c {
		n.LastChild = c.PrevSibling
	}
	if c.PrevSibling != nil {
		c.PrevSibling.NextSibling = c.NextSibling
	}
}

const encPrefix = "charset="

func encoding(node *html.Node) string {
	if node.Type != html.ElementNode || node.Data != "meta" {
		return ""
	}
	for _, a := range node.Attr {
		if a.Key == "charset" {
			return a.Val
		}
		if a.Key != "content" {
			continue
		}
		if i := strings.Index(a.Val, encPrefix); i >= 0 {
			return a.Val[i+len(encPrefix):]
		}
	}
	return ""
}

// Extract extracts title and main content from HTML.
func Extract(rd io.Reader) (string, string, error) {
	return extract(rd, &url.URL{})
}

// Base is like Extract, but base is used to complete URL of links.
func Base(rd io.Reader, base string) (string, string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", "", err
	}
	return extract(rd, u)
}

func extract(rd io.Reader, base *url.URL) (string, string, error) {
	// FIXME: improve.
	// use machine learning.
	// consider length of text.

	doc, err := html.Parse(rd)
	if err != nil {
		return "", "", err
	}
	enc, title, nodes := parse(doc, base)
	if len(nodes) == 0 {
		return title, "", nil
	}
	if len(nodes) == 1 {
		doc = nodes[0].Parent
	} else {
		var commonAncestor *html.Node
	loop:
		for f, s, i := nodes[0].Parent, nodes[1].Parent, 0; i < len(nodes)-1; f, i = commonAncestor, i+1 {
			s = nodes[i+1].Parent
			for c := f; c != nil; c = c.Parent {
				for c2 := s; c2 != nil; c2 = c2.Parent {
					if c == c2 {
						commonAncestor = c
						continue loop
					}
				}
			}
		}
		doc = commonAncestor
	}
	var buf bytes.Buffer
	html.Render(&buf, doc)
	content := convertString(&buf, enc)
	title = convertString(strings.NewReader(title), enc)
	return title, content, nil
}

func isIgnoreItself(n *html.Node) bool {
	if _, toIgnoreItself := tagNamesToIgnoreOnlyItself[n.Data]; n.Type == html.ElementNode && toIgnoreItself {
		return true
	}
	return false
}

func weightByClassID(n *html.Node) int {
	var weight int
	for _, a := range n.Attr {
		if a.Key == "class" || a.Key == "id" {
			for _, pat := range positivePattern {
				if indexWord(a.Val, pat) >= 0 {
					weight++
				}
			}
			for _, pat := range negativePattern {
				if indexWord(a.Val, pat) >= 0 {
					weight--
				}
			}
		}
	}
	return weight
}

func parse(n *html.Node, base *url.URL) (enc, title string, nodes []*html.Node) {
	p := parser{base: base}
	return p.parse(n, 0, make([]*html.Node, 0, 8))
}

type parser struct {
	level int
	base  *url.URL
}

func (p *parser) parse(n *html.Node, prelevel int, levelSet []*html.Node) (enc, title string, nodes []*html.Node) {
	enc, title = scanHead(n)

	level := prelevel
	nodes = levelSet

	_, toIgnore := tagNamesToIgnore[n.Data]
	if (n.Type == html.ElementNode && toIgnore) || (n.Type == html.CommentNode) {
		removeChild(n.Parent, n)
		return
	}
	if n.Type == html.ElementNode && !toIgnore && !isIgnoreItself(n) {
		weight := weightByClassID(n)
		if weight < 0 {
			removeChild(n.Parent, n)
			return
		}
		if weight > 0 {
			level++
		}
		cleanAttribute(n, p.base)
	}
	if n.Type == html.TextNode {
		if level > p.level {
			nodes = []*html.Node{n}
		}
		if level == p.level {
			nodes = append(nodes, n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e, t, n := p.parse(c, level, nodes)
		if e != "" {
			enc = e
		}
		if t != "" {
			title = t
		}
		nodes = n
	}
	if level > p.level {
		p.level = level
	}
	if n.Type == html.ElementNode && n.Data == "div" && n.FirstChild == nil {
		removeChild(n.Parent, n)
	}
	return
}

func scanHead(n *html.Node) (enc, title string) {
	if n.Type != html.ElementNode || n.Data != "head" {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if e := encoding(c); e != "" {
			enc = e
		}
		switch {
		case c.Type != html.ElementNode, c.Data != "title", c.FirstChild == nil:
			continue
		}
		title = c.FirstChild.Data
		break
	}
	return enc, title
}

func cleanAttribute(n *html.Node, base *url.URL) {
	switch n.Data {
	case "a":
		for _, a := range n.Attr {
			if a.Key == "href" {
				n.Attr = []html.Attribute{{Namespace: a.Namespace, Key: a.Key, Val: a.Val}}
				break
			}
		}
	case "img":
		attr := make([]html.Attribute, 0, 1)
		for _, a := range n.Attr {
			val := a.Val
			switch a.Key {
			case "src":
				u, err := url.Parse(val)
				if err != nil {
					return
				}
				val = base.ResolveReference(u).String()
				fallthrough
			case "alt", "width", "height":
				attr = append(attr, html.Attribute{Namespace: a.Namespace, Key: a.Key, Val: val})
			}
		}
		n.Attr = attr
	default:
		n.Attr = nil
	}
}

func convert(inStream io.Reader, outStream io.Writer, enc string) error {
	var render io.Reader
	switch strings.ToLower(enc) {
	case "euc-jp":
		render = transform.NewReader(inStream, japanese.EUCJP.NewDecoder())
	case "shift_jis":
		render = transform.NewReader(inStream, japanese.ShiftJIS.NewDecoder())
	default:
		render = inStream
	}

	_, err := io.Copy(outStream, render)
	return err
}

func convertString(rd io.Reader, enc string) string {
	var buf bytes.Buffer
	err := convert(rd, &buf, enc)
	if err != nil {
		return ""
	}
	b, err := ioutil.ReadAll(&buf)
	if err != nil {
		return ""
	}
	return string(b)
}
