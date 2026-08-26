package xmlq

import (
	"bytes"
	"encoding/xml"
)

type contentKind int

const (
	contentEmpty contentKind = iota
	contentLeaf
	contentElements
	contentMixed
)

type formatter struct {
	out       *bytes.Buffer
	prefix    string
	indent    string
	masks     []Mask
	path      []xml.Name
	depth     int
	lineStart bool
}

func (f *formatter) pretty() bool {
	return f.indent != "" || f.prefix != ""
}

func (f *formatter) writeIndent() {
	if !f.pretty() {
		return
	}
	if !f.lineStart {
		f.out.WriteByte('\n')
	}
	f.out.WriteString(f.prefix)
	for i := 0; i < f.depth; i++ {
		f.out.WriteString(f.indent)
	}
	f.lineStart = false
}

func (f *formatter) nodes(nodes []node) {
	for _, n := range nodes {
		if n.kind == nodeText && isAllSpace(n.data) {
			continue
		}
		f.formatNode(n)
	}
}

func (f *formatter) formatNode(n node) {
	switch n.kind {
	case nodeXMLDecl, nodePI, nodeComment, nodeDirective:
		f.writeIndent()
		f.out.Write(n.raw)
		f.lineStart = false
	case nodeElement:
		f.formatElement(n)
	case nodeText:
		f.writeIndent()
		f.writeText(n)
	case nodeCDATA:
		f.writeCDATA(n, false)
	}
}

func (f *formatter) formatElement(n node) {
	f.writeIndent()
	f.writeStart(n, n.selfClosing)
	if n.selfClosing {
		return
	}

	f.path = append(f.path, n.xmlName())
	defer func() {
		f.path = f.path[:len(f.path)-1]
	}()

	switch classify(n) {
	case contentEmpty:
		f.writeEnd(n)
	case contentLeaf:
		for _, c := range n.children {
			switch c.kind {
			case nodeText:
				f.writeText(c)
			case nodeCDATA:
				f.writeCDATA(c, true)
			case nodeXMLDecl, nodePI, nodeComment, nodeDirective, nodeElement:
				f.formatInline(c)
			}
		}
		f.writeEnd(n)
	case contentElements:
		f.depth++
		for _, c := range n.children {
			if c.kind == nodeText && isAllSpace(c.data) {
				continue
			}
			f.formatNode(c)
		}
		f.depth--
		f.writeIndent()
		f.writeEnd(n)
	case contentMixed:
		for _, c := range n.children {
			f.formatInline(c)
		}
		f.writeEnd(n)
	}
}

func (f *formatter) formatInline(n node) {
	switch n.kind {
	case nodeText:
		f.writeText(n)
	case nodeCDATA:
		f.writeCDATA(n, true)
	case nodeElement:
		f.writeStart(n, n.selfClosing)
		if n.selfClosing {
			return
		}
		f.path = append(f.path, n.xmlName())
		for _, c := range n.children {
			f.formatInline(c)
		}
		f.path = f.path[:len(f.path)-1]
		f.writeEnd(n)
	case nodeXMLDecl, nodePI, nodeComment, nodeDirective:
		f.out.Write(n.raw)
		f.lineStart = false
	}
}

func (f *formatter) writeStart(n node, empty bool) {
	f.out.WriteByte('<')
	writeName(f.out, n.prefix, n.local)
	for _, a := range n.attrs {
		f.out.WriteByte(' ')
		f.out.Write(a.name)
		f.out.WriteByte('=')
		q := a.quote
		if q != '"' && q != '\'' {
			q = '"'
		}
		f.out.WriteByte(q)
		f.out.Write(a.rawValue)
		f.out.WriteByte(q)
	}
	if empty {
		f.out.WriteString("/>")
	} else {
		f.out.WriteByte('>')
	}
	f.lineStart = false
}

func (f *formatter) writeEnd(n node) {
	f.out.WriteString("</")
	writeName(f.out, n.prefix, n.local)
	f.out.WriteByte('>')
	f.lineStart = false
}

func writeName(buf *bytes.Buffer, prefix, local string) {
	if prefix != "" {
		buf.WriteString(prefix)
		buf.WriteByte(':')
	}
	buf.WriteString(local)
}

func (f *formatter) writeText(n node) {
	mask := findMask(f.path, f.masks)
	if mask == nil || len(bytes.TrimSpace(n.data)) == 0 {
		f.out.Write(n.rawData)
		f.lineStart = false
		return
	}
	masked := applyMask(xml.CharData(n.data), mask)
	f.out.Write(escapeText([]byte(masked)))
	f.lineStart = false
}

func (f *formatter) writeCDATA(n node, inline bool) {
	if inner, ok := parseAsXML(n.data); ok {
		if !inline {
			f.writeIndent()
		}
		f.out.WriteString("<![CDATA[")
		if f.pretty() {
			f.out.WriteByte('\n')
		}
		innerF := formatter{
			out:       f.out,
			prefix:    f.prefix,
			indent:    f.indent,
			masks:     f.masks,
			path:      append([]xml.Name(nil), f.path...),
			depth:     f.depth + 1,
			lineStart: true,
		}
		if !f.pretty() {
			innerF.depth = 0
			innerF.lineStart = false
		}
		innerF.nodes(inner)
		if f.pretty() {
			f.lineStart = innerF.lineStart
			if !inline {
				f.writeIndent()
			}
		}
		f.out.WriteString("]]>")
		f.lineStart = false
		return
	}

	content := n.data
	if mask := findMask(f.path, f.masks); mask != nil && len(bytes.TrimSpace(content)) > 0 {
		content = []byte(applyMask(xml.CharData(content), mask))
	}
	if !inline {
		f.writeIndent()
	}
	f.out.WriteString("<![CDATA[")
	f.out.Write(content)
	f.out.WriteString("]]>")
	f.lineStart = false
}

func classify(n node) contentKind {
	hasElem := false
	hasNonWSText := false
	hasBlockCDATA := false
	hasLeafCDATA := false
	hasMisc := false
	for _, c := range n.children {
		switch c.kind {
		case nodeElement:
			hasElem = true
		case nodeText:
			if !isAllSpace(c.data) {
				hasNonWSText = true
			}
		case nodeCDATA:
			if _, ok := parseAsXML(c.data); ok {
				hasBlockCDATA = true
			} else if len(bytes.TrimSpace(c.data)) > 0 {
				hasLeafCDATA = true
			}
		case nodeComment, nodePI, nodeDirective, nodeXMLDecl:
			hasMisc = true
		}
	}
	if !hasElem && !hasNonWSText && !hasBlockCDATA && !hasLeafCDATA && !hasMisc {
		return contentEmpty
	}
	if hasBlockCDATA && !hasElem && !hasNonWSText {
		return contentElements
	}
	if (hasElem || hasMisc) && !hasNonWSText && !hasLeafCDATA {
		return contentElements
	}
	if !hasElem && !hasBlockCDATA && !hasMisc {
		return contentLeaf
	}
	return contentMixed
}

func escapeText(b []byte) []byte {
	if bytes.IndexByte(b, '&') < 0 && bytes.IndexByte(b, '<') < 0 {
		return b
	}
	var out bytes.Buffer
	out.Grow(len(b) + 8)
	for _, c := range b {
		switch c {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		default:
			out.WriteByte(c)
		}
	}
	return out.Bytes()
}
