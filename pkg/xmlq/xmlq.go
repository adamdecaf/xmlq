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
				depthLevel -= 1
				if !insideStartTag && !insideEndTag {
					buf.WriteString("\n")
				}
				buf.WriteString("</")
				insideStartTag = false
				insideEndTag = false

			case '?':
				// XML header, do nothing
				buf.WriteString("<?")
				// insideEndTag = true
				insideStartTag = true

			case ' ':
				buf.WriteString(fmt.Sprintf("S "))
				if insideStartTag {
					buf.Write(data[i : i+1])
				}

			default:
				// start tag
				// If we're already inside a start tag move that to a new line
				// buf.WriteString(fmt.Sprintf("-insideStartTag=%v  depthLevel=%d-", insideStartTag, depthLevel))
				// buf.WriteString("B\n")
				insideStartTag = true
				depthLevel += 1

				// insideStartTag = true
				// insideEndTag = false
				// buf.WriteString("\n")
				// if insideStartTag {

				// buf.WriteString(fmt.Sprintf("- insideStartTag=%v  insideEndTag=%v  depthLevel=%d-", insideStartTag, insideEndTag, depthLevel))
				buf.WriteString("\n")

				padding := fmt.Sprintf("%s%s", options.Prefix, strings.Repeat(options.Indent, depthLevel))
				buf.WriteString(padding)

				// depthLevel += 1
				buf.WriteString("<")
				buf.Write(data[i : i+1])
			}

		case '/':
			buf.WriteString("/")

		case '?': // xml header
			buf.WriteString("?")
			depthLevel -= 1
			insideStartTag = true

		case '>':
			buf.WriteString(">")
			if insideEndTag {
				// buf.WriteString("A\n")
				insideEndTag = false
			}

		case '\n', '\r':
			// skip newlines since we add them

		case ' ':
			// Only write spaces that are inside of elements
			// buf.WriteString(fmt.Sprintf("- insideStartTag=%v  insideEndTag=%v  depthLevel=%d-", insideStartTag, insideEndTag, depthLevel))
			// if insideStartTag {
			// 	buf.WriteString("S")
			// 	buf.Write(data[i : i+1])
			// }

		default:
			buf.Write(data[i : i+1])
		}
	}
	goto read

done:

	_ = insideStartTag
	_ = insideEndTag
	_ = doubleNewLine

	return buf.Bytes(), nil
}

func applyMaskIfNeeded(elm xml.Name, input string) string {
	return input
}
