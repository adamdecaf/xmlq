package xmlq

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"slices"
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
	ShowLastFour  MaskingType = "show-last-four"
	ShowMiddle    MaskingType = "show-middle"
	ShowWordStart MaskingType = "show-word-start"
	ShowNone      MaskingType = "show-none"
)

func process(d *xml.Decoder, e *xml.Encoder, maskStack *[]*Mask, options *Options) error {
	for {
		token, err := d.RawToken()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			mask := findMask(t, options.Masks)
			*maskStack = append(*maskStack, mask)

			startCopy := t
			if startCopy.Name.Space != "" {
				startCopy.Name.Local = fmt.Sprintf("%s:%s", startCopy.Name.Space, startCopy.Name.Local)
				startCopy.Name.Space = ""
			}
			if err := e.EncodeToken(startCopy); err != nil {
				return err
			}

		case xml.CharData:
			elm := t
			start, end, middle := []byte("<"), []byte(">"), []byte("><")
			if bytes.HasPrefix(elm, start) && bytes.HasSuffix(elm, end) && bytes.Contains(elm, middle) {
				// Recurse to process inner XML with same encoder for proper indentation
				innerD := xml.NewDecoder(bytes.NewReader(elm))
				if err := process(innerD, e, maskStack, options); err != nil {
					return fmt.Errorf("rendering inner xml: %w", err)
				}
			} else {
				if len(*maskStack) > 0 {
					m := (*maskStack)[len(*maskStack)-1]
					elm = applyMask(elm, m)
				}
				if err := e.EncodeToken(elm); err != nil {
					return err
				}
			}

		case xml.EndElement:
			endCopy := t
			if endCopy.Name.Space != "" {
				endCopy.Name.Local = fmt.Sprintf("%s:%s", endCopy.Name.Space, endCopy.Name.Local)
				endCopy.Name.Space = ""
			}
			if err := e.EncodeToken(endCopy); err != nil {
				return err
			}

			if len(*maskStack) > 0 {
				*maskStack = (*maskStack)[:len(*maskStack)-1]
			}

		default:
			if err := e.EncodeToken(t); err != nil {
				return err
			}
		}

		if err := e.Flush(); err != nil {
			return err
		}
	}
}

// MarshalIndent will unmarshal and remarshal XML with specified element values masked.
// Masking matches on element names and applying the masking logic specified.
//
// The XML is remarshaled with indentation while preserving the original structure,
// including namespace prefixes. Self-closing elements are expanded by the XML decoder/encoder.
func MarshalIndent(input io.Reader, opts *Options) ([]byte, error) {
	var options Options
	if opts != nil {
		options = *opts
	} else {
		options.Indent = "  "
	}

	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	e.Indent(options.Prefix, options.Indent)

	d := xml.NewDecoder(input)

	var maskStack []*Mask
	if err := process(d, e, &maskStack, &options); err != nil {
		return nil, err
	}

	if err := e.Close(); err != nil {
		return nil, err
	}

	// Clean up excessive newlines
	out := bytes.ReplaceAll(buf.Bytes(), []byte("\n\n"), []byte("\n"))

	return out, nil
}

func findMask(start xml.StartElement, masks []Mask) *Mask {
	for i := range masks {
		namesMatch := strings.EqualFold(start.Name.Local, masks[i].Name)

		spacesMatch := true
		if start.Name.Space != "" && masks[i].Space != "" {
			spacesMatch = strings.EqualFold(start.Name.Space, masks[i].Space)
		}

		if namesMatch && spacesMatch {
			return &masks[i]
		}
	}
	return nil
}

func applyMask(elm xml.CharData, mask *Mask) xml.CharData {
	if mask == nil {
		return elm
	}

	switch mask.Mask {
	case ShowLastFour:
		if len(elm) < 5 {
			return xml.CharData(bytes.Repeat([]byte("*"), len(elm)))
		}
		return xml.CharData(append(
			bytes.Repeat([]byte("*"), len(elm)-4),
			elm[len(elm)-4:]...,
		))

	case ShowMiddle:
		if len(elm) < 2 {
			return xml.CharData(bytes.Repeat([]byte("*"), len(elm)))
		}
		quarter := (len(elm) / 4) + 1
		if len(elm) == 4 {
			quarter = 1
		}
		return xml.CharData(
			slices.Concat(
				bytes.Repeat([]byte("*"), quarter),
				elm[quarter:len(elm)-quarter],
				bytes.Repeat([]byte("*"), quarter),
			),
		)

	case ShowWordStart:
		if len(elm) == 0 {
			return xml.CharData(nil)
		}

		fields := strings.Fields(string(elm))
		var out string
		for i := range fields {
			if len(fields[i]) > 0 {
				out += fields[i][0:1] + strings.Repeat("*", len(fields[i])-1)
			}
			if i < len(fields)-1 {
				out += " "
			}
		}
		return xml.CharData([]byte(out))

	case ShowNone:
		return xml.CharData([]byte(strings.Repeat("*", len(elm))))

	}
	return elm
}
