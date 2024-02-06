package xmlq

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type Options struct {
	Prefix, Indent string

	Masks []Mask
}

type Mask struct {
	Name, Space string

	Mask MaskingType
}

type MaskingType string

var (
	ShowLastFour MaskingType = "show-last-four"
	ShowMiddle   MaskingType = "show-middle"
	ShowNone     MaskingType = "show-none"
)

// maxTokenLimit is the upper limit of how many XML tokens to consume.
const maxTokenLimit = 1000

func MarshalIndent(input io.Reader, opts *Options) ([]byte, error) {
	var buf bytes.Buffer

	depthLevel := -1
	var tokensConsumed int
	var previousAction xml.Token

	var options Options
	if opts != nil {
		options = *opts
	} else {
		opts.Indent = "  "
	}

	decoder := xml.NewDecoder(input)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		tokensConsumed += 1
		if tokensConsumed >= maxTokenLimit {
			return nil, fmt.Errorf("depth of xml document is too deep at %d levels", tokensConsumed)
		}
		if token == nil {
			continue
		}

		switch elm := token.(type) {
		case xml.StartElement:
			depthLevel += 1

			// Write a newline after two StartElements are encountered in a row
			if _, ok := previousAction.(xml.StartElement); ok {
				buf.WriteString("\n")
			}

			padding := fmt.Sprintf("%s%s", options.Prefix, strings.Repeat(options.Indent, depthLevel))
			innerElement := elm.Name.Local
			if elm.Name.Space != "" {
				innerElement = fmt.Sprintf("%s:%s", elm.Name.Space, elm.Name.Local)
			}
			buf.WriteString(fmt.Sprintf("%s<%s>", padding, innerElement))

			// Save token for next iteration
			previousAction = elm

		case xml.CharData:
			var value string
			switch prev := previousAction.(type) {
			case xml.StartElement:
				value = applyMaskIfNeeded(prev.Name, string(elm))

			case xml.EndElement:
				value = applyMaskIfNeeded(prev.Name, string(elm))
			}
			buf.WriteString(strings.TrimSpace(value))

		case xml.EndElement:
			innerElement := elm.Name.Local
			if elm.Name.Space != "" {
				innerElement = fmt.Sprintf("%s:%s", elm.Name.Space, elm.Name.Local)
			}

			buf.WriteString(fmt.Sprintf("</%s>\n", innerElement))

			// Save token for next iteration
			previousAction = elm
			depthLevel -= 1
		}
	}

	return buf.Bytes(), nil
}

func applyMaskIfNeeded(elm xml.Name, input string) string {
	return input
}
