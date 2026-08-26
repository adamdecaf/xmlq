package xmlq

import (
	"bytes"
	"io"
)

type Options struct {
	Prefix, Indent string

	Masks []Mask
}

type Mask struct {
	// Name is the element to mask. A single local name ("Id") matches any
	// element with that name. A path ("DbtrAcct/Id") matches when the last
	// segment is the current element and each earlier segment appears as an
	// ancestor, in order. Intermediate elements may be omitted, so
	// "DbtrAcct/Id" matches both DbtrAcct/Id and DbtrAcct/Id/Othr/Id, but
	// not Rpt/Id. Segments may include a prefix ("ct:Id").
	Name, Space string

	Mask MaskingType
}

type MaskingType string

var (
	ShowLastFour  MaskingType = "show-last-four"
	ShowMiddle    MaskingType = "show-middle"
	ShowWordStart MaskingType = "show-word-start"
	ShowNone      MaskingType = "show-none"
)

// MarshalIndent pretty-prints XML and applies any configured element masks.
//
// The document's structure is preserved: namespace prefixes and declarations,
// attribute names and values, self-closing tags, comments, processing
// instructions, and CDATA versus parsed text are not rewritten. Only
// insignificant whitespace between elements is changed, and only character
// data of elements that match a mask is replaced.
//
// Well-formed XML inside a CDATA section is pretty-printed and masked inside
// that CDATA section. Escaped markup in ordinary text is left as text.
func MarshalIndent(input io.Reader, opts *Options) ([]byte, error) {
	var options Options
	if opts != nil {
		options = *opts
	} else {
		options.Indent = "  "
	}

	src, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	src = bytes.TrimPrefix(src, []byte{0xEF, 0xBB, 0xBF})

	doc, err := parse(src)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	f := &formatter{
		out:       &buf,
		prefix:    options.Prefix,
		indent:    options.Indent,
		masks:     options.Masks,
		lineStart: true,
	}
	f.nodes(doc)
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		buf.WriteByte('\n')
		out = buf.Bytes()
	}
	return out, nil
}
