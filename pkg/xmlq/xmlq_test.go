package xmlq

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

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

func fixtureMasks() []Mask {
	return []Mask{
		{Name: "from", Mask: ShowMiddle},
		{Name: "Id", Mask: ShowLastFour},
		{Name: "Nm", Mask: ShowWordStart},
		{Name: "StrtNm", Mask: ShowWordStart},
	}
}

func marshal(t *testing.T, path, expected string) {
	t.Helper()

	fd, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { fd.Close() })

	output, err := MarshalIndent(fd, &Options{
		Indent: "  ",
		Masks:  fixtureMasks(),
	})
	require.NoError(t, err)

	bs, err := os.ReadFile(expected)
	require.NoError(t, err)
	bs = bytes.ReplaceAll(bs, []byte("\r\n"), []byte("\n"))

	require.Equal(t, string(bs), string(output))
}

func TestPathMasking(t *testing.T) {
	input := strings.TrimSpace(`
<Document>
  <DbtrAcct>
    <Id>
      <Othr>
        <Id>11000179512199001</Id>
      </Othr>
    </Id>
  </DbtrAcct>
  <CdtrAcct>
    <Id>
      <Othr>
        <Id>12000194212199001</Id>
      </Othr>
    </Id>
  </CdtrAcct>
  <Rpt>
    <Id>stmt-2026-001</Id>
  </Rpt>
  <MktPrctc>
    <Id>frb.fednow.01</Id>
  </MktPrctc>
</Document>
`)

	output, err := MarshalIndent(strings.NewReader(input), &Options{
		Indent: "  ",
		Masks: []Mask{
			{Name: "DbtrAcct/Id", Mask: ShowLastFour},
			{Name: "Rpt/Id", Mask: ShowNone},
		},
	})
	require.NoError(t, err)

	got := string(output)
	require.Contains(t, got, "<Id>*************9001</Id>")
	require.Contains(t, got, "<Id>12000194212199001</Id>")
	require.Contains(t, got, "<Id>*************</Id>")
	require.Contains(t, got, "<Id>frb.fednow.01</Id>")
	require.NotContains(t, got, "11000179512199001")
	require.NotContains(t, got, "stmt-2026-001")
}

func TestStructurePreserved(t *testing.T) {
	must := func(t *testing.T, input string, masks []Mask) string {
		t.Helper()
		out, err := MarshalIndent(strings.NewReader(input), &Options{Indent: "  ", Masks: masks})
		require.NoError(t, err)
		require.True(t, utf8.Valid(out), "output is not valid UTF-8")
		return string(out)
	}

	t.Run("xmlns prefixes", func(t *testing.T) {
		got := must(t, `<root xmlns:ct="urn:ct"><ct:Id>11000179512199001</ct:Id></root>`, []Mask{
			{Name: "Id", Mask: ShowLastFour},
		})
		require.Contains(t, got, `xmlns:ct="urn:ct"`)
		require.Contains(t, got, `<ct:Id>*************9001</ct:Id>`)
		require.NotContains(t, got, "_xmlns")
		require.NotContains(t, got, `xmlns:_`)
	})

	t.Run("xml:lang", func(t *testing.T) {
		got := must(t, `<p xml:lang="en">hi</p>`, nil)
		require.Contains(t, got, `xml:lang="en"`)
		require.NotContains(t, got, `_xml`)
	})

	t.Run("xsi:type", func(t *testing.T) {
		got := must(t, `<Amt xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:decimal">1.00</Amt>`, nil)
		require.Contains(t, got, `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`)
		require.Contains(t, got, `xsi:type="xsd:decimal"`)
		require.NotContains(t, got, `_xmlns`)
		require.NotContains(t, got, `xmlns:xsi="xsi"`)
	})

	t.Run("self-closing", func(t *testing.T) {
		got := must(t, `<root><br/></root>`, nil)
		require.Contains(t, got, `<br/>`)
		require.NotContains(t, got, `<br></br>`)
	})

	t.Run("empty pair stays a pair", func(t *testing.T) {
		got := must(t, `<root><br></br></root>`, nil)
		require.Contains(t, got, `<br></br>`)
		require.NotContains(t, got, `<br/>`)
	})

	t.Run("escaped markup stays text", func(t *testing.T) {
		got := must(t, `<note><body>&lt;p&gt;hello&lt;/p&gt;&lt;p&gt;world&lt;/p&gt;</body></note>`, nil)
		require.Contains(t, got, `&lt;p&gt;hello&lt;/p&gt;&lt;p&gt;world&lt;/p&gt;`)
		require.NotContains(t, got, "<p>")
	})

	t.Run("almost-xml text does not fail", func(t *testing.T) {
		got := must(t, `<x>&lt;notxml&gt;&gt;&lt;also&gt;</x>`, nil)
		require.Contains(t, got, `&lt;notxml&gt;&gt;&lt;also&gt;`)
	})

	t.Run("apostrophe and gt kept", func(t *testing.T) {
		got := must(t, `<note><body>Don't forget</body><n>5 &gt; 3</n><m>5 > 3</m></note>`, nil)
		require.Contains(t, got, `Don't forget`)
		require.NotContains(t, got, `&#39;`)
		require.Contains(t, got, `5 &gt; 3`)
		require.Contains(t, got, `5 > 3`)
	})

	t.Run("spaces in text kept", func(t *testing.T) {
		got := must(t, `<extra>  Preserve spaces on both sides  </extra>`, nil)
		require.Contains(t, got, `<extra>  Preserve spaces on both sides  </extra>`)
	})

	t.Run("parent mask skips indent", func(t *testing.T) {
		got := must(t, "<Nm>\n  <First>John</First>\n</Nm>", []Mask{
			{Name: "Nm", Mask: ShowNone},
		})
		require.Contains(t, got, "<First>John</First>")
		require.NotContains(t, got, "*")
	})

	t.Run("utf8 last four", func(t *testing.T) {
		got := must(t, `<n>José€</n>`, []Mask{{Name: "n", Mask: ShowLastFour}})
		require.Contains(t, got, `<n>*osé€</n>`)
	})

	t.Run("single-quoted attributes", func(t *testing.T) {
		got := must(t, `<a b='c'><d e="f"/></a>`, nil)
		require.Contains(t, got, `b='c'`)
		require.Contains(t, got, `e="f"`)
	})

	t.Run("comment and pi", func(t *testing.T) {
		got := must(t, `<?xml-stylesheet type="text/xsl" href="a.xsl"?><root><!-- keep --><x/></root>`, nil)
		require.Contains(t, got, `<?xml-stylesheet type="text/xsl" href="a.xsl"?>`)
		require.Contains(t, got, `<!-- keep -->`)
	})

	t.Run("default indent when opts nil", func(t *testing.T) {
		out, err := MarshalIndent(strings.NewReader(`<a><b>1</b></a>`), nil)
		require.NoError(t, err)
		require.Equal(t, "<a>\n  <b>1</b>\n</a>\n", string(out))
	})

	t.Run("compact when indent empty", func(t *testing.T) {
		out, err := MarshalIndent(strings.NewReader(`<a><b>1</b></a>`), &Options{})
		require.NoError(t, err)
		require.Equal(t, "<a><b>1</b></a>\n", string(out))
	})
}

func TestCDATA(t *testing.T) {
	must := func(t *testing.T, input string, masks []Mask) string {
		t.Helper()
		out, err := MarshalIndent(strings.NewReader(input), &Options{Indent: "  ", Masks: masks})
		require.NoError(t, err)
		return string(out)
	}

	t.Run("multi element stays cdata", func(t *testing.T) {
		got := must(t, `<note><body><![CDATA[<p>hello</p><p>world</p>]]></body></note>`, nil)
		require.Contains(t, got, "<![CDATA[")
		require.Contains(t, got, "]]>")
		require.Contains(t, got, "<p>hello</p>")
		require.Contains(t, got, "<p>world</p>")
		// Pretty-printed inside CDATA, not promoted to siblings of body.
		require.Regexp(t, `(?s)<body>\s*<!\[CDATA\[.*<p>hello</p>.*\]\]>\s*</body>`, got)
		require.NotRegexp(t, `\n[ \t]*\n[ \t]*]]>`, got)
	})

	t.Run("single element stays cdata", func(t *testing.T) {
		got := must(t, `<note><body><![CDATA[<p>hello</p>]]></body></note>`, nil)
		require.Contains(t, got, "<![CDATA[")
		require.Contains(t, got, "<p>hello</p>")
		require.NotContains(t, got, `&lt;p&gt;`)
	})

	t.Run("pretty printed inner xml", func(t *testing.T) {
		got := must(t, "<note><body><![CDATA[\n  <p>hello</p>\n  <p>world</p>\n]]></body></note>", nil)
		require.Contains(t, got, "<![CDATA[")
		require.Contains(t, got, "<p>hello</p>")
	})

	t.Run("inner xml is masked", func(t *testing.T) {
		got := must(t, `<x><![CDATA[<root><Id>11000179512199001</Id><Nm>John Doe</Nm></root>]]></x>`, []Mask{
			{Name: "Id", Mask: ShowLastFour},
			{Name: "Nm", Mask: ShowWordStart},
		})
		require.Contains(t, got, "<![CDATA[")
		require.Contains(t, got, "<Id>*************9001</Id>")
		require.Contains(t, got, "<Nm>J*** D**</Nm>")
		require.NotContains(t, got, "11000179512199001")
		require.NotContains(t, got, "John Doe")
	})

	t.Run("opaque cdata is masked as text", func(t *testing.T) {
		got := must(t, `<secret><![CDATA[token-value]]></secret>`, []Mask{
			{Name: "secret", Mask: ShowNone},
		})
		require.Contains(t, got, "<![CDATA[***********]]>")
	})

	t.Run("xml declaration stays inside cdata", func(t *testing.T) {
		got := must(t, `<x><![CDATA[<?xml version="1.0"?><a>1</a><b>2</b>]]></x>`, nil)
		require.Regexp(t, `(?s)<x>\s*<!\[CDATA\[.*<\?xml version="1.0"\?>.*<a>1</a>.*<b>2</b>.*\]\]>\s*</x>`, got)
	})

	t.Run("not xml stays opaque", func(t *testing.T) {
		got := must(t, `<x><![CDATA[just some text]]></x>`, nil)
		require.Contains(t, got, "<![CDATA[just some text]]>")
	})
}

func TestPrettyPrintPreservesShape(t *testing.T) {
	files := []string{
		filepath.Join("testdata", "note.xml"),
		filepath.Join("testdata", "pacs_008.xml"),
		filepath.Join("testdata", "admi_002.xml"),
		filepath.Join("testdata", "pacs_028.xml"),
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			in, err := os.ReadFile(path)
			require.NoError(t, err)
			out, err := MarshalIndent(bytes.NewReader(in), &Options{Indent: "  "})
			require.NoError(t, err)

			before, err := parse(in)
			require.NoError(t, err)
			after, err := parse(out)
			require.NoError(t, err)
			require.NoError(t, sameShape(before, after))
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		`<a><b></a></b>`,
		`<a>`,
		`<!-- unterminated`,
		`<![CDATA[unterminated`,
		`<a b=1/>`,
		`<a>&unknown;</a>`,
		`</a>`,
	}
	for _, in := range cases {
		_, err := MarshalIndent(strings.NewReader(in), nil)
		require.Error(t, err, in)
	}
}

func TestBOMAndMisc(t *testing.T) {
	t.Run("utf8 bom", func(t *testing.T) {
		in := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`<a>1</a>`)...)
		out, err := MarshalIndent(bytes.NewReader(in), &Options{Indent: "  "})
		require.NoError(t, err)
		require.Equal(t, "<a>1</a>\n", string(out))
	})

	t.Run("doctype", func(t *testing.T) {
		in := `<!DOCTYPE note [<!ELEMENT note (#PCDATA)>]><note>hi</note>`
		out, err := MarshalIndent(strings.NewReader(in), &Options{Indent: "  "})
		require.NoError(t, err)
		require.Contains(t, string(out), `<!DOCTYPE note [<!ELEMENT note (#PCDATA)>]>`)
		require.Contains(t, string(out), `<note>hi</note>`)
	})

	t.Run("char ref preserved when unmasked", func(t *testing.T) {
		out, err := MarshalIndent(strings.NewReader(`<n>&#169;abc</n>`), &Options{Indent: "  "})
		require.NoError(t, err)
		require.Equal(t, "<n>&#169;abc</n>\n", string(out))
	})

	t.Run("masked text escapes lt", func(t *testing.T) {
		out, err := MarshalIndent(strings.NewReader(`<n>ab&lt;cd</n>`), &Options{
			Indent: "  ",
			Masks:  []Mask{{Name: "n", Mask: ShowLastFour}},
		})
		require.NoError(t, err)
		require.Equal(t, "<n>*b&lt;cd</n>\n", string(out))
	})

	t.Run("prefix", func(t *testing.T) {
		out, err := MarshalIndent(strings.NewReader(`<a><b>1</b></a>`), &Options{Prefix: "|", Indent: "  "})
		require.NoError(t, err)
		require.Equal(t, "|<a>\n|  <b>1</b>\n|</a>\n", string(out))
	})
}

func TestMixedContent(t *testing.T) {
	got, err := MarshalIndent(strings.NewReader(`<p>Hello <b>world</b>!</p>`), &Options{Indent: "  "})
	require.NoError(t, err)
	require.Equal(t, "<p>Hello <b>world</b>!</p>\n", string(got))
}

func TestMixedContentMasking(t *testing.T) {
	input := `<p>Hello <b>secret</b> world</p>`

	t.Run("parent none leaves child text", func(t *testing.T) {
		got, err := MarshalIndent(strings.NewReader(input), &Options{
			Indent: "  ",
			Masks:  []Mask{{Name: "p", Mask: ShowNone}},
		})
		require.NoError(t, err)
		require.Equal(t, "<p>******<b>secret</b>******</p>\n", string(got))
		require.NotContains(t, string(got), "<b>*")
		require.NotContains(t, string(got), "<*>")
	})

	t.Run("parent word start keeps child and spacing", func(t *testing.T) {
		got, err := MarshalIndent(strings.NewReader(input), &Options{
			Indent: "  ",
			Masks:  []Mask{{Name: "p", Mask: ShowWordStart}},
		})
		require.NoError(t, err)
		require.Equal(t, "<p>H**** <b>secret</b> w****</p>\n", string(got))
	})

	t.Run("child mask does not hide sibling text", func(t *testing.T) {
		got, err := MarshalIndent(strings.NewReader(input), &Options{
			Indent: "  ",
			Masks:  []Mask{{Name: "b", Mask: ShowNone}},
		})
		require.NoError(t, err)
		require.Equal(t, "<p>Hello <b>******</b> world</p>\n", string(got))
	})
}

func TestMaskedEntities(t *testing.T) {
	must := func(t *testing.T, input string, mask Mask) string {
		t.Helper()
		out, err := MarshalIndent(strings.NewReader(input), &Options{
			Indent: "  ",
			Masks:  []Mask{mask},
		})
		require.NoError(t, err)
		return string(out)
	}

	t.Run("ampersand is decoded before masking", func(t *testing.T) {
		// Decoded "John&Doe" is 8 runes. The raw entity form is 12 bytes.
		require.Equal(t, "<n>********</n>\n", must(t, `<n>John&amp;Doe</n>`, Mask{Name: "n", Mask: ShowNone}))
		require.Equal(t, "<n>J*******</n>\n", must(t, `<n>John&amp;Doe</n>`, Mask{Name: "n", Mask: ShowWordStart}))
	})

	t.Run("hex char refs are decoded", func(t *testing.T) {
		// &#x41;..&#x45; is ABCDE, not the entity source text.
		require.Equal(t, "<n>*BCDE</n>\n", must(t, `<n>&#x41;&#x42;&#x43;&#x44;&#x45;</n>`, Mask{Name: "n", Mask: ShowLastFour}))
		require.Equal(t, "<n>****</n>\n", must(t, `<n>&#xA9;abc</n>`, Mask{Name: "n", Mask: ShowNone}))
	})

	t.Run("apos and quot are decoded", func(t *testing.T) {
		require.Equal(t, "<n>O******</n>\n", must(t, `<n>O&apos;Brien</n>`, Mask{Name: "n", Mask: ShowWordStart}))
		require.Equal(t, "<n>******</n>\n", must(t, `<n>say&quot;hi</n>`, Mask{Name: "n", Mask: ShowNone}))
	})
}

func TestPrefixedPathMasking(t *testing.T) {
	input := strings.TrimSpace(`
<root xmlns:ct="urn:ct" xmlns:mr="urn:mr">
  <ct:DbtrAcct>
    <ct:Id>11000179512199001</ct:Id>
  </ct:DbtrAcct>
  <mr:DbtrAcct>
    <mr:Id>12000194212199001</mr:Id>
  </mr:DbtrAcct>
</root>
`)

	output, err := MarshalIndent(strings.NewReader(input), &Options{
		Indent: "  ",
		Masks: []Mask{
			{Name: "ct:DbtrAcct/ct:Id", Mask: ShowLastFour},
		},
	})
	require.NoError(t, err)

	got := string(output)
	require.Contains(t, got, `xmlns:ct="urn:ct"`)
	require.Contains(t, got, `xmlns:mr="urn:mr"`)
	require.Contains(t, got, "<ct:Id>*************9001</ct:Id>")
	require.Contains(t, got, "<mr:Id>12000194212199001</mr:Id>")
	require.NotContains(t, got, "11000179512199001")
	require.NotContains(t, got, "_xmlns")
}

func sameShape(a, b []node) error {
	a = dropSpace(a)
	b = dropSpace(b)
	if len(a) != len(b) {
		return fmt.Errorf("node count %d != %d", len(a), len(b))
	}
	for i := range a {
		if err := sameNode(a[i], b[i]); err != nil {
			return err
		}
	}
	return nil
}

func sameNode(a, b node) error {
	if a.kind != b.kind {
		return fmt.Errorf("kind %v != %v", a.kind, b.kind)
	}
	switch a.kind {
	case nodeElement:
		if a.prefix != b.prefix || a.local != b.local || a.selfClosing != b.selfClosing {
			return fmt.Errorf("element <%s/> mismatch: %+v vs %+v", displayName(a.prefix, a.local), a, b)
		}
		if len(a.attrs) != len(b.attrs) {
			return fmt.Errorf("<%s> attr count %d != %d", displayName(a.prefix, a.local), len(a.attrs), len(b.attrs))
		}
		for i := range a.attrs {
			if !bytes.Equal(a.attrs[i].name, b.attrs[i].name) || !bytes.Equal(a.attrs[i].rawValue, b.attrs[i].rawValue) {
				return fmt.Errorf("<%s> attr %s=%q vs %s=%q", displayName(a.prefix, a.local),
					a.attrs[i].name, a.attrs[i].rawValue, b.attrs[i].name, b.attrs[i].rawValue)
			}
		}
		return sameShape(a.children, b.children)
	case nodeText:
		if !bytes.Equal(a.data, b.data) {
			return fmt.Errorf("text %q != %q", a.data, b.data)
		}
	case nodeCDATA:
		if innerA, okA := parseAsXML(a.data); okA {
			innerB, okB := parseAsXML(b.data)
			if !okB {
				return fmt.Errorf("CDATA was XML, output is not")
			}
			return sameShape(innerA, innerB)
		}
		if !bytes.Equal(bytes.TrimSpace(a.data), bytes.TrimSpace(b.data)) {
			return fmt.Errorf("CDATA %q != %q", a.data, b.data)
		}
	case nodeXMLDecl, nodePI, nodeComment, nodeDirective:
		if !bytes.Equal(a.raw, b.raw) {
			return fmt.Errorf("raw %q != %q", a.raw, b.raw)
		}
	}
	return nil
}

func dropSpace(nodes []node) []node {
	var out []node
	for _, n := range nodes {
		if n.kind == nodeText && isAllSpace(n.data) {
			continue
		}
		if n.kind == nodeElement {
			n.children = dropSpace(n.children)
		}
		out = append(out, n)
	}
	return out
}
