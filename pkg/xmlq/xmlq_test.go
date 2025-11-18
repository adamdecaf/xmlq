package xmlq

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalIndent(t *testing.T) {
	t.Run("note.xml", func(t *testing.T) {
		marshal(t, filepath.Join("testdata", "note.xml"), filepath.Join("testdata", "note.expected.xml"))
	})

	t.Run("pacs_008.xml", func(t *testing.T) {
		marshal(t, filepath.Join("testdata", "pacs_008.xml"), filepath.Join("testdata", "pacs_008.expected.xml"))
	})

	t.Run("admi.002", func(t *testing.T) {
		marshal(t, filepath.Join("testdata", "admi_002.xml"), filepath.Join("testdata", "admi_002.expected.xml"))
	})

	t.Run("pacs_028", func(t *testing.T) {
		marshal(t, filepath.Join("testdata", "pacs_028.xml"), filepath.Join("testdata", "pacs_028.expected.xml"))
	})
}

func marshal(t *testing.T, path, expected string) {
	t.Helper()

	fd, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { fd.Close() })

	output, err := MarshalIndent(fd, &Options{
		Indent: "  ",
		Masks: []Mask{
			// found in notes.xml
			{
				Name: "from",
				Mask: ShowMiddle,
			},
			// found in pacs_008.xml
			{
				// <ct:Id>11000179512199001</ct:Id>
				Name: "Id",
				Mask: ShowLastFour,
			},
			{
				// <ct:Nm>John Doe</ct:Nm>
				Name: "Nm",
				Mask: ShowWordStart,
			},
			{
				// <ct:StrtNm>123 Any St</ct:StrtNm>
				Name: "StrtNm",
				Mask: ShowWordStart,
			},
		},
	})
	require.NoError(t, err)

	bs, err := os.ReadFile(expected)
	require.NoError(t, err)

	// Strip windows quotes
	bs = bytes.ReplaceAll(bs, []byte("\r\n"), []byte("\n"))

	require.Equal(t, string(bs), string(output))
}

func TestMasking(t *testing.T) {
	t.Run("find mask", func(t *testing.T) {
		masks := []Mask{
			{Name: "Id", Space: "", Mask: ShowNone},
			{Name: "StrtNm", Space: "", Mask: ShowWordStart},
			{Name: "Nm", Space: "ct", Mask: ShowWordStart},
		}

		elm := xml.StartElement{Name: xml.Name{Local: "Id"}}
		expected := &masks[0]
		output := findMask(elm, masks)
		require.Equal(t, expected, output)

		elm = xml.StartElement{Name: xml.Name{Local: "StrtNm"}}
		expected = &masks[1]
		output = findMask(elm, masks)
		require.Equal(t, expected, output)

		elm = xml.StartElement{Name: xml.Name{Local: "StrtNm", Space: "ct"}}
		expected = &masks[1]
		output = findMask(elm, masks)
		require.Equal(t, expected, output)

		elm = xml.StartElement{Name: xml.Name{Local: "Nm"}}
		expected = &masks[2]
		output = findMask(elm, masks)
		require.Equal(t, expected, output)

		elm = xml.StartElement{Name: xml.Name{Local: "Nm", Space: "ct"}}
		expected = &masks[2]
		output = findMask(elm, masks)
		require.Equal(t, expected, output)
	})

	t.Run("last four", func(t *testing.T) {
		cases := []struct {
			input, expected string
		}{
			{"", ""},
			{" ", "*"},
			{"  ", "**"},
			{"123", "***"},
			{"1234", "****"},
			{
				input:    "12345",
				expected: "*2345",
			},
			{
				input:    "123456",
				expected: "**3456",
			},
			{
				input:    "Adam Shannon",
				expected: "********nnon",
			},
		}

		for i := range cases {
			output := applyMask(xml.CharData(cases[i].input), &Mask{Mask: ShowLastFour})
			require.Equal(t, cases[i].expected, string(output), fmt.Sprintf("input: %q", cases[i].input))
		}
	})

	t.Run("middle", func(t *testing.T) {
		cases := []struct {
			input, expected string
		}{
			{"", ""},
			{" ", "*"},
			{"  ", "**"},
			{"123", "*2*"},
			{
				input:    "1234",
				expected: "*23*",
			},
			{
				input:    "12345",
				expected: "**3**",
			},
			{
				input:    "123456",
				expected: "**34**",
			},
			{
				input:    "Adam Shannon",
				expected: "**** Sha****",
			},
		}

		for i := range cases {
			output := applyMask(xml.CharData(cases[i].input), &Mask{Mask: ShowMiddle})
			require.Equal(t, cases[i].expected, string(output), fmt.Sprintf("input: %q", cases[i].input))
		}
	})

	t.Run("word start", func(t *testing.T) {
		cases := []struct {
			input, expected string
		}{
			{"", ""},
			{" ", ""},
			{"  ", ""},
			{"123", "1**"},
			{
				input:    "1 2 3",
				expected: "1 2 3",
			},
			{
				input:    "12 34 56",
				expected: "1* 3* 5*",
			},
			{
				input:    "123 456",
				expected: "1** 4**",
			},
			{
				input:    "Adam Shannon",
				expected: "A*** S******",
			},
		}

		for i := range cases {
			output := applyMask(xml.CharData(cases[i].input), &Mask{Mask: ShowWordStart})
			require.Equal(t, cases[i].expected, string(output), fmt.Sprintf("input: %q", cases[i].input))
		}
	})

	t.Run("none", func(t *testing.T) {
		cases := []struct {
			input, expected string
		}{
			{"", ""},
			{" ", "*"},
			{"  ", "**"},
			{"123", "***"},
			{
				input:    "1 2 3",
				expected: "*****",
			},
			{
				input:    "12 34 56",
				expected: "********",
			},
			{
				input:    "123 456",
				expected: "*******",
			},
			{
				input:    "Adam Shannon",
				expected: "************",
			},
		}

		for i := range cases {
			output := applyMask(xml.CharData(cases[i].input), &Mask{Mask: ShowNone})
			require.Equal(t, cases[i].expected, string(output), fmt.Sprintf("input: %q", cases[i].input))
		}
	})
}
