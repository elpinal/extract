package extract

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var tagNamesToIgnore = map[string]int{
	//"a":      0,
	//"body":   0,
	//"html":   0,
	//"svg":    0,

	"embed":    0,
	"form":     0,
	"iframe":   0,
	"object":   0,
	"option":   0,
	"script":   0,
	"style":    0,
	"meta":     0,
	"link":     0,
	"head":     0,
	"aside":    0,
	"noscript": 0,
}

var tagNamesToIgnoreOnlyItself = map[string]int{
	"html": 0,
	"body": 0,
}

var negativeRegexp = regexp.MustCompile("breadcrumb|combx|comment|contact|disqus|foot|footer|footnote|hidden|link|media|meta|mod-conversations|pager|pagination|promo|reaction|related|scroll|share|shoutbox|sidebar|social|sponsor|tags|toolbox|widget")
var positiveRegexp = regexp.MustCompile(`article|body|content|entry|hentry|page\b|post|text`)

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
	/*
		c.Parent = nil
		c.PrevSibling = nil
		c.NextSibling = nil
	*/
}

func nth(nodes []*html.Node, n int) *html.Node {
	if n >= len(nodes) {
		return &html.Node{Parent: nil}
	}
	return nodes[n]
}

func encoding(node *html.Node) string {
	if node.Type == html.ElementNode && node.Data == "meta" {
		for _, a := range node.Attr {
			if a.Key == "charset" {
				return a.Val
			}
			if i := strings.Index(a.Val, "charset="); a.Key == "content" && i >= 0 {
				return a.Val[i+8:]
			}
		}
	}
	return ""
}

// FIXME: improve.
// use machine learning.
// consider length of text.
// instead of []byte, s should be io.Reader, and FromURL should be FromString.
func Extract(s []byte) (string, string, error) {
	r := bytes.NewReader(s)
	doc, err := html.Parse(r)
	if err != nil {
		return "", "", err
	}
	var title, enc string
	var level int
	var maxLevel int
	var levelSet = make(map[int][]*html.Node)
	var f func(*html.Node)
	f = func(n *html.Node) {
		var preLevel = level
		var ignoreItself bool
		if _, toIgnoreItself := tagNamesToIgnoreOnlyItself[n.Data]; n.Type == html.ElementNode && toIgnoreItself {
			ignoreItself = true
		}

		if n.Type == html.ElementNode && n.Data == "head" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if e := encoding(c); e != "" {
					enc = e
				}
				if c.Type == html.ElementNode && c.Data == "title" {
					title = c.FirstChild.Data
					break
				}
			}
		}

		if n.Type == html.ElementNode && n.Data == "title" {
			title = n.FirstChild.Data
		} else if n.Type == html.ElementNode && n.Data == "meta" {
			if e := encoding(n); e != "" {
				enc = e
			}
		} else if _, toIgnore := tagNamesToIgnore[n.Data]; n.Type == html.ElementNode && !toIgnore && !ignoreItself {
			var classIDWeight int
			for _, a := range n.Attr {
				if a.Key == "class" || a.Key == "id" {
					if s := positiveRegexp.FindString(a.Val); s != "" {
						classIDWeight++
					}
					if s := negativeRegexp.FindString(a.Val); s != "" {
						classIDWeight--
					}
				}
			}
			if classIDWeight >= 0 {
				if classIDWeight > 0 {
					level++
				}
				if n.Data == "a" {
					for _, a := range n.Attr {
						if a.Key == "href" {
							n.Attr = []html.Attribute{html.Attribute{Namespace: a.Namespace, Key: a.Key, Val: a.Val}}
							break
						}
					}
				} else if n.Data == "img" {
					attr := make([]html.Attribute, 0, 1)
					for _, a := range n.Attr {
						switch a.Key {
						case "src", "alt", "width", "height":
							attr = append(attr, html.Attribute{Namespace: a.Namespace, Key: a.Key, Val: a.Val})
						}
					}
					n.Attr = attr
				} else {
					n.Attr = nil
				}
			} else {
				removeChild(n.Parent, n)
				return
			}
		} else if (n.Type == html.ElementNode && toIgnore) || (n.Type == html.CommentNode) {
			removeChild(n.Parent, n)
			return
		}
		if n.Type == html.TextNode {
			levelSet[level] = append(levelSet[level], n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		if level > maxLevel {
			maxLevel = level
		}
		level = preLevel
		if n.Type == html.ElementNode && n.Data == "div" && n.FirstChild == nil {
			removeChild(n.Parent, n)
		}
	}
	f(doc)
	if nodes := levelSet[maxLevel]; len(nodes) == 0 {
		return "", "", fmt.Errorf("%v", "There are no content")
	} else if len(nodes) == 1 {
		doc = nodes[0].Parent
	} else {
		var commonAncestor *html.Node
	loop:
		for f, s, i := nodes[0].Parent, nodes[1].Parent, 0; i < len(nodes)-1; f, s, i = commonAncestor, nth(nodes, i+2).Parent, i+1 {
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
	var b bytes.Buffer
	html.Render(&b, doc)
	content := conversionString(&b, enc)
	title = conversionString(strings.NewReader(title), enc)
	return title, content, nil
}

func FromURL(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	return Extract(body)
}

func conversion(inStream io.Reader, outStream io.Writer, enc string) error {
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

func conversionString(rd io.Reader, enc string) string {
	var bf bytes.Buffer
	err := conversion(rd, &bf, enc)
	if err != nil {
		return ""
	}
	bt, err := ioutil.ReadAll(&bf)
	if err != nil {
		return ""
	}
	return string(bt)
}
