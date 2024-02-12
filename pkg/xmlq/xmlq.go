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

// MarshalIndent will unmarshal and remarshal XML with specified element values masked.
// Masking matches on element names and applying the masking logic specified.
//
// The XML is not remarshaled exactly as it arrives.
//   - Self-closing elements are expanded into start and end tags.
//   - Namespace prefixes are generally preserved, but namespace attributes may appear in the output.
//   - Certain characters are escaped with their xml entity values.
func MarshalIndent(input io.Reader, opts *Options) ([]byte, error) {
	var options Options
	if opts != nil {
		options = *opts
	} else {
		opts.Indent = "  "
	}

	var buf bytes.Buffer

	d := xml.NewDecoder(input)
	e := xml.NewEncoder(&buf)
	e.Indent(options.Prefix, options.Indent)

	var applicableMask *Mask

	for {
		token, err := d.RawToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// process token
		switch t := token.(type) {
		case xml.ProcInst:
			e.EncodeToken(t)
			e.Flush()
			buf.WriteString("\n")

		case xml.StartElement:
			// Look for a mask to apply later on
			applicableMask = findMask(t, options.Masks)
			if t.Name.Space != "" {
				t.Name.Local = fmt.Sprintf("%s:%s", t.Name.Space, t.Name.Local)
				t.Name.Space = ""
			}
			e.EncodeToken(t)

		case xml.CharData:
			elm := applyMask(t, applicableMask)
			e.EncodeToken(elm)
			applicableMask = nil

		case xml.EndElement:
			if t.Name.Space != "" {
				t.Name.Local = fmt.Sprintf("%s:%s", t.Name.Space, t.Name.Local)
				t.Name.Space = ""
			}
			e.EncodeToken(t)

		default:
			e.EncodeToken(t)
		}

		err = e.Flush()
		if err != nil {
			return nil, fmt.Errorf("flushing tokens: %w", err)
		}
	}

	// Remove duplicate newlines
	out := bytes.ReplaceAll(buf.Bytes(), []byte("\n  \n  "), []byte("\n  "))
	out = bytes.ReplaceAll(out, []byte("\n\n"), []byte("\n"))

	return out, e.Close()
}

func findMask(start xml.StartElement, masks []Mask) *Mask {
	for i := range masks {
		namesMatch := strings.EqualFold(start.Name.Local, masks[i].Name)

		spacesMatch := true // default to assuming namespaces match
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

	// We have a mask.MaskingType
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
		// We want to show less than half of the content
		quarter := (len(elm) / 4) + 1
		if len(elm) == 4 {
			quarter = 1 // special case length of four to show middle two
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
			out += fields[i][0:1] + strings.Repeat("*", len(fields[i])-1)
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
