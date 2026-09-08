package util

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var allowedElements = map[string]bool{
	"p": true, "br": true, "hr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"b": true, "strong": true, "i": true, "em": true, "u": true, "s": true, "del": true, "strike": true,
	"mark": true, "sub": true, "sup": true,
	"span": true, "div": true, "blockquote": true, "pre": true, "code": true,
	"ul": true, "ol": true, "li": true,
	"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "th": true, "td": true,
	"colgroup": true, "col": true,
	"img": true, "a": true, "video": true, "source": true, "audio": true,
}

var dangerousTags = map[string]bool{
	"script": true, "iframe": true, "frame": true, "frameset": true,
	"object": true, "embed": true, "applet": true, "style": true,
	"link": true, "meta": true, "base": true, "form": true,
	"input": true, "button": true, "textarea": true, "select": true,
	"option": true, "noscript": true, "svg": true, "math": true,
}

// CleanHTML sanitizes an HTML fragment against an allowlist, preserving RichEditor structure
// while stripping scripts, event handlers, and dangerous protocols.
func CleanHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	for _, node := range nodes {
		for _, cleaned := range sanitizeNode(node) {
			if err := html.Render(&buf, cleaned); err != nil {
				continue
			}
		}
	}
	return buf.String()
}

// Each returned node is detached; stripping a wrapper can produce multiple nodes.
func sanitizeNode(n *html.Node) []*html.Node {
	switch n.Type {
	case html.TextNode:
		return []*html.Node{{
			Type: html.TextNode,
			Data: n.Data,
		}}

	case html.ElementNode:
		tag := strings.ToLower(n.Data)
		if dangerousTags[tag] {
			return nil // Drop entire node and children
		}
		if !allowedElements[tag] {
			// For unrecognized tags, strip tag but keep allowed children
			var children []*html.Node
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				children = append(children, sanitizeNode(c)...)
			}
			return children
		}

		newNode := &html.Node{
			Type:     html.ElementNode,
			Data:     tag,
			DataAtom: n.DataAtom,
		}

		// Filter attributes
		var attrs []html.Attribute
		for _, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			if strings.HasPrefix(key, "on") {
				continue // Strip event handlers
			}
			val := strings.TrimSpace(attr.Val)

			if isAllowedAttr(tag, key) {
				if isURLAttr(key) {
					if !isSafeURL(val, tag == "img") {
						continue
					}
				} else if key == "style" {
					if !isSafeStyle(val) {
						continue
					}
				}
				attrs = append(attrs, html.Attribute{
					Namespace: attr.Namespace,
					Key:       key,
					Val:       val,
				})
			}
		}
		newNode.Attr = attrs

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			for _, cleanedChild := range sanitizeNode(c) {
				newNode.AppendChild(cleanedChild)
			}
		}
		return []*html.Node{newNode}

	case html.CommentNode:
		return nil // Drop comments

	default:
		return nil
	}
}

func isAllowedAttr(tag, attr string) bool {
	switch attr {
	case "id", "class", "style", "title":
		return true
	case "href", "target", "rel":
		return tag == "a"
	case "src", "alt", "width", "height":
		return tag == "img" || tag == "video" || tag == "audio" || tag == "source"
	case "type", "controls", "autoplay", "loop", "muted", "poster":
		return tag == "video" || tag == "audio" || tag == "source"
	case "colspan", "rowspan", "align", "valign":
		return tag == "table" || tag == "tr" || tag == "td" || tag == "th" || tag == "col" || tag == "colgroup"
	default:
		return false
	}
}

func isURLAttr(attr string) bool {
	return attr == "href" || attr == "src" || attr == "poster"
}

func isSafeURL(url string, allowDataImage bool) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	// Remove control characters and whitespace
	cleaned := ""
	for _, r := range lower {
		if r > 32 && r < 127 {
			cleaned += string(r)
		}
	}

	if strings.HasPrefix(cleaned, "javascript:") ||
		strings.HasPrefix(cleaned, "vbscript:") ||
		strings.HasPrefix(cleaned, "data:text/html") {
		return false
	}

	if allowDataImage && (strings.HasPrefix(cleaned, "data:image/png") ||
		strings.HasPrefix(cleaned, "data:image/jpeg") ||
		strings.HasPrefix(cleaned, "data:image/gif") ||
		strings.HasPrefix(cleaned, "data:image/webp")) {
		return true
	}

	return strings.HasPrefix(cleaned, "http://") ||
		strings.HasPrefix(cleaned, "https://") ||
		strings.HasPrefix(cleaned, "//") ||
		strings.HasPrefix(cleaned, "/") ||
		strings.HasPrefix(cleaned, "./") ||
		strings.HasPrefix(cleaned, "#")
}

func isSafeStyle(style string) bool {
	lower := strings.ToLower(style)
	return !strings.Contains(lower, "javascript:") &&
		!strings.Contains(lower, "expression(") &&
		!strings.Contains(lower, "behavior:") &&
		!strings.Contains(lower, "-moz-binding")
}
