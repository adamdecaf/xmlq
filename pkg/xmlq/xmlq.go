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

func MarshalIndent(input io.Reader, opts *Options) ([]byte, error) {
	var depthLevel int

	var options Options
	if opts != nil {
		options = *opts
	} else {
		opts.Indent = "  "
	}

	var buf bytes.Buffer
	data := make([]byte, 32)
	size := len(data)

	var insideStartTag bool
	var insideEndTag bool
	var doubleNewLine bool

read:
	n, err := io.ReadFull(input, data)
	if err == io.EOF {
		goto done
	}
	size = n

	// < is the last character, so return it to the input
	if data[size-1] == '<' {
		size -= 1
		input = io.MultiReader(strings.NewReader("<"), input)
	}

	for i := 0; i < size; i++ {
		switch data[i] {
		case '<':
			if i+1 >= size {
				// There's nothing else read in the buffer
				panic("bad thing")
			}

			// Check what tag is coming
			i += 1
			switch data[i] {
			case '/':
				// end tag
				insideStartTag = false
				insideEndTag = true
				buf.WriteString("</")

			case '?':
				// XML header, do nothing
				insideEndTag = true
				buf.WriteString("<?")

			default:
				// start tag
				if insideStartTag {
					doubleNewLine = doubleNewLine && (insideStartTag && !insideEndTag)
					if !doubleNewLine {
						buf.WriteString("\n")

						padding := fmt.Sprintf("%s%s", options.Prefix, strings.Repeat(options.Indent, depthLevel))
						if padding != "" {
							buf.WriteString(padding)
						}
					}
				}
				insideStartTag = true
				depthLevel += 1
				buf.WriteString("<")
				buf.Write(data[i : i+1])
			}

		case '/':
			buf.WriteString("/")
			insideStartTag = false
			insideEndTag = true

		case '>':
			buf.WriteString(">")
			if !insideStartTag && insideEndTag {
				buf.WriteString("\n")

				depthLevel -= 1
				if depthLevel < 0 {
					depthLevel = 0
				}

				padding := fmt.Sprintf("%s%s", options.Prefix, strings.Repeat(options.Indent, depthLevel))
				if padding != "" {
					buf.WriteString(padding)
				}
			}
			insideEndTag = false

		case '\n', '\r':
			// skip newlines since we add them

		default:
			buf.Write(data[i : i+1])
		}
	}
	goto read

done:
	return buf.Bytes(), nil
}

func applyMaskIfNeeded(elm xml.Name, input string) string {
	return input
}
