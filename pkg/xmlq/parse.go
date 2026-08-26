package xmlq

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"unicode/utf8"
)

type nodeKind int

const (
	nodeXMLDecl nodeKind = iota
	nodePI
	nodeComment
	nodeDirective
	nodeElement
	nodeText
	nodeCDATA
)

type node struct {
	kind        nodeKind
	prefix      string
	local       string
	attrs       []attr
	selfClosing bool
	children    []node
	raw         []byte // original bytes for decls, comments, PIs, directives
	data        []byte // decoded text or CDATA content
	rawData     []byte // original text bytes, entities intact
}

type attr struct {
	name     []byte
	prefix   string
	local    string
	quote    byte
	rawValue []byte
}

func (n node) xmlName() xml.Name {
	return xml.Name{Space: n.prefix, Local: n.local}
}

func parse(src []byte) ([]node, error) {
	p := &parser{s: src, line: 1, col: 1}
	var nodes []node
	for !p.eof() {
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

type parser struct {
	s    []byte
	i    int
	line int
	col  int
}

func (p *parser) eof() bool { return p.i >= len(p.s) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.s[p.i]
}

func (p *parser) hasPrefix(s string) bool {
	return bytes.HasPrefix(p.s[p.i:], []byte(s))
}

func (p *parser) next() byte {
	if p.eof() {
		return 0
	}
	b := p.s[p.i]
	p.i++
	if b == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return b
}

func (p *parser) err(msg string) error {
	return fmt.Errorf("xmlq: line %d, column %d: %s", p.line, p.col, msg)
}

func (p *parser) skipSpace() {
	for !p.eof() && isXMLSpace(p.peek()) {
		p.next()
	}
}

func isXMLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func (p *parser) parseNode() (node, error) {
	if p.peek() != '<' {
		return p.parseText()
	}
	switch {
	case p.hasPrefix("<?"):
		return p.parsePI()
	case p.hasPrefix("<!--"):
		return p.parseComment()
	case p.hasPrefix("<![CDATA["):
		return p.parseCDATA()
	case p.hasPrefix("<!"):
		return p.parseDirective()
	case p.hasPrefix("</"):
		return node{}, p.err("unexpected end tag")
	default:
		return p.parseElement()
	}
}

func (p *parser) parseText() (node, error) {
	start := p.i
	for !p.eof() && p.peek() != '<' {
		p.next()
	}
	raw := p.s[start:p.i]
	data, err := unescape(raw)
	if err != nil {
		return node{}, p.err(err.Error())
	}
	return node{kind: nodeText, data: data, rawData: raw}, nil
}

func (p *parser) parsePI() (node, error) {
	start := p.i
	if !p.hasPrefix("<?") {
		return node{}, p.err("expected processing instruction")
	}
	p.i += 2
	p.col += 2
	nameStart := p.i
	if _, _, _, err := p.readName(); err != nil {
		return node{}, err
	}
	name := p.s[nameStart:p.i]
	for !p.eof() && !p.hasPrefix("?>") {
		p.next()
	}
	if !p.hasPrefix("?>") {
		return node{}, p.err("unclosed processing instruction")
	}
	p.i += 2
	p.col += 2
	raw := append([]byte(nil), p.s[start:p.i]...)
	kind := nodePI
	if bytes.Equal(name, []byte("xml")) {
		kind = nodeXMLDecl
	}
	return node{kind: kind, raw: raw}, nil
}

func (p *parser) parseComment() (node, error) {
	start := p.i
	if !p.hasPrefix("<!--") {
		return node{}, p.err("expected comment")
	}
	p.i += 4
	p.col += 4
	for !p.eof() && !p.hasPrefix("-->") {
		if p.hasPrefix("--") {
			return node{}, p.err("comment contains '--'")
		}
		p.next()
	}
	if !p.hasPrefix("-->") {
		return node{}, p.err("unclosed comment")
	}
	p.i += 3
	p.col += 3
	return node{kind: nodeComment, raw: append([]byte(nil), p.s[start:p.i]...)}, nil
}

func (p *parser) parseCDATA() (node, error) {
	if !p.hasPrefix("<![CDATA[") {
		return node{}, p.err("expected CDATA")
	}
	p.i += 9
	p.col += 9
	start := p.i
	for !p.eof() && !p.hasPrefix("]]>") {
		p.next()
	}
	if !p.hasPrefix("]]>") {
		return node{}, p.err("unclosed CDATA")
	}
	data := append([]byte(nil), p.s[start:p.i]...)
	p.i += 3
	p.col += 3
	return node{kind: nodeCDATA, data: data}, nil
}

func (p *parser) parseDirective() (node, error) {
	start := p.i
	if p.next() != '<' || p.next() != '!' {
		return node{}, p.err("expected directive")
	}
	depth := 0
	var quote byte
	for !p.eof() {
		b := p.peek()
		if quote != 0 {
			p.next()
			if b == quote {
				quote = 0
			}
			continue
		}
		switch b {
		case '"', '\'':
			quote = b
			p.next()
		case '[':
			depth++
			p.next()
		case ']':
			if depth > 0 {
				depth--
			}
			p.next()
		case '>':
			if depth == 0 {
				p.next()
				return node{kind: nodeDirective, raw: append([]byte(nil), p.s[start:p.i]...)}, nil
			}
			p.next()
		default:
			p.next()
		}
	}
	return node{}, p.err("unclosed directive")
}

func (p *parser) parseElement() (node, error) {
	if p.next() != '<' {
		return node{}, p.err("expected start tag")
	}
	_, prefix, local, err := p.readName()
	if err != nil {
		return node{}, err
	}
	if local == "" {
		return node{}, p.err("missing element name")
	}

	n := node{kind: nodeElement, prefix: prefix, local: local}

	for {
		p.skipSpace()
		if p.eof() {
			return node{}, p.err("unclosed start tag")
		}
		if p.peek() == '>' {
			p.next()
			break
		}
		if p.peek() == '/' {
			p.next()
			if p.peek() != '>' {
				return node{}, p.err("expected '/>'")
			}
			p.next()
			n.selfClosing = true
			return n, nil
		}
		a, err := p.parseAttr()
		if err != nil {
			return node{}, err
		}
		n.attrs = append(n.attrs, a)
		if !p.eof() && !isXMLSpace(p.peek()) && p.peek() != '/' && p.peek() != '>' {
			return node{}, p.err("expected space between attributes")
		}
	}

	for {
		if p.eof() {
			return node{}, p.err(fmt.Sprintf("unclosed element <%s>", displayName(prefix, local)))
		}
		if p.hasPrefix("</") {
			if err := p.parseEndTag(prefix, local); err != nil {
				return node{}, err
			}
			return n, nil
		}
		child, err := p.parseNode()
		if err != nil {
			return node{}, err
		}
		n.children = append(n.children, child)
	}
}

func (p *parser) parseEndTag(prefix, local string) error {
	if !p.hasPrefix("</") {
		return p.err("expected end tag")
	}
	p.i += 2
	p.col += 2
	_, endPrefix, endLocal, err := p.readName()
	if err != nil {
		return err
	}
	p.skipSpace()
	if p.peek() != '>' {
		return p.err("expected '>' in end tag")
	}
	p.next()
	if endPrefix != prefix || endLocal != local {
		return p.err(fmt.Sprintf("end tag </%s> does not match <%s>",
			displayName(endPrefix, endLocal), displayName(prefix, local)))
	}
	return nil
}

func (p *parser) parseAttr() (attr, error) {
	rawName, prefix, local, err := p.readName()
	if err != nil {
		return attr{}, err
	}
	p.skipSpace()
	if p.peek() != '=' {
		return attr{}, p.err("expected '=' in attribute")
	}
	p.next()
	p.skipSpace()
	quote := p.peek()
	if quote != '"' && quote != '\'' {
		return attr{}, p.err("expected quoted attribute value")
	}
	p.next()
	valStart := p.i
	for !p.eof() && p.peek() != quote {
		p.next()
	}
	if p.peek() != quote {
		return attr{}, p.err("unclosed attribute value")
	}
	rawValue := append([]byte(nil), p.s[valStart:p.i]...)
	p.next()
	return attr{
		name:     append([]byte(nil), rawName...),
		prefix:   prefix,
		local:    local,
		quote:    quote,
		rawValue: rawValue,
	}, nil
}

func (p *parser) readName() (raw []byte, prefix, local string, err error) {
	start := p.i
	if p.eof() || !isNameStart(p.peek()) {
		return nil, "", "", p.err("expected name")
	}
	p.next()
	for !p.eof() && isNameChar(p.peek()) {
		p.next()
	}
	raw = p.s[start:p.i]
	prefix, local = splitQName(raw)
	return raw, prefix, local, nil
}

func splitQName(raw []byte) (prefix, local string) {
	if i := bytes.IndexByte(raw, ':'); i >= 0 {
		return string(raw[:i]), string(raw[i+1:])
	}
	return "", string(raw)
}

func displayName(prefix, local string) string {
	if prefix == "" {
		return local
	}
	return prefix + ":" + local
}

func isNameStart(b byte) bool {
	return b == ':' || b == '_' ||
		(b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		b >= 0x80
}

func isNameChar(b byte) bool {
	return isNameStart(b) || b == '-' || b == '.' || (b >= '0' && b <= '9')
}

func unescape(b []byte) ([]byte, error) {
	if bytes.IndexByte(b, '&') < 0 {
		return append([]byte(nil), b...), nil
	}
	var out bytes.Buffer
	out.Grow(len(b))
	i := 0
	for i < len(b) {
		if b[i] != '&' {
			out.WriteByte(b[i])
			i++
			continue
		}
		semi := bytes.IndexByte(b[i:], ';')
		if semi < 0 {
			return nil, fmt.Errorf("unterminated entity")
		}
		ent := b[i : i+semi+1]
		switch {
		case bytes.Equal(ent, []byte("&lt;")):
			out.WriteByte('<')
		case bytes.Equal(ent, []byte("&gt;")):
			out.WriteByte('>')
		case bytes.Equal(ent, []byte("&amp;")):
			out.WriteByte('&')
		case bytes.Equal(ent, []byte("&apos;")):
			out.WriteByte('\'')
		case bytes.Equal(ent, []byte("&quot;")):
			out.WriteByte('"')
		case bytes.HasPrefix(ent, []byte("&#x")) || bytes.HasPrefix(ent, []byte("&#X")):
			r, err := parseCharRef(ent[3:len(ent)-1], 16)
			if err != nil {
				return nil, err
			}
			out.WriteRune(r)
		case bytes.HasPrefix(ent, []byte("&#")):
			r, err := parseCharRef(ent[2:len(ent)-1], 10)
			if err != nil {
				return nil, err
			}
			out.WriteRune(r)
		default:
			return nil, fmt.Errorf("unknown entity %s", ent)
		}
		i += semi + 1
	}
	return out.Bytes(), nil
}

func parseCharRef(digits []byte, base int) (rune, error) {
	if len(digits) == 0 {
		return 0, fmt.Errorf("empty character reference")
	}
	v, err := strconv.ParseInt(string(digits), base, 32)
	if err != nil || v < 0 || v > unicodeMax || !utf8.ValidRune(rune(v)) {
		return 0, fmt.Errorf("invalid character reference")
	}
	return rune(v), nil
}

const unicodeMax = 0x10FFFF

func isAllSpace(b []byte) bool {
	for _, c := range b {
		if !isXMLSpace(c) {
			return false
		}
	}
	return true
}

func hasElement(nodes []node) bool {
	for _, n := range nodes {
		if n.kind == nodeElement {
			return true
		}
	}
	return false
}

func parseAsXML(data []byte) ([]node, bool) {
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '<' {
		return nil, false
	}
	doc, err := parse(data)
	if err != nil || !hasElement(doc) {
		return nil, false
	}
	return doc, true
}
