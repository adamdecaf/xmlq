package xmlq

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindMask(t *testing.T) {
	t.Run("local name", func(t *testing.T) {
		masks := []Mask{
			{Name: "Id", Space: "", Mask: ShowNone},
			{Name: "StrtNm", Space: "", Mask: ShowWordStart},
			{Name: "Nm", Space: "ct", Mask: ShowWordStart},
		}

		path := []xml.Name{{Local: "Id"}}
		require.Equal(t, &masks[0], findMask(path, masks))

		path = []xml.Name{{Local: "StrtNm"}}
		require.Equal(t, &masks[1], findMask(path, masks))

		path = []xml.Name{{Local: "StrtNm", Space: "ct"}}
		require.Equal(t, &masks[1], findMask(path, masks))

		path = []xml.Name{{Local: "Nm"}}
		require.Equal(t, &masks[2], findMask(path, masks))

		path = []xml.Name{{Local: "Nm", Space: "ct"}}
		require.Equal(t, &masks[2], findMask(path, masks))
	})

	t.Run("path", func(t *testing.T) {
		masks := []Mask{
			{Name: "DbtrAcct/Id", Mask: ShowLastFour},
			{Name: "Rpt/Id", Mask: ShowNone},
		}

		dbtrLeaf := []xml.Name{
			{Local: "CdtTrfTxInf"},
			{Local: "DbtrAcct"},
			{Local: "Id"},
			{Local: "Othr"},
			{Local: "Id"},
		}
		require.Equal(t, &masks[0], findMask(dbtrLeaf, masks))

		dbtrWrapper := []xml.Name{
			{Local: "DbtrAcct"},
			{Local: "Id"},
		}
		require.Equal(t, &masks[0], findMask(dbtrWrapper, masks))

		rpt := []xml.Name{
			{Local: "BkToCstmrStmt"},
			{Local: "Rpt"},
			{Local: "Id"},
		}
		require.Equal(t, &masks[1], findMask(rpt, masks))

		otherId := []xml.Name{
			{Local: "MktPrctc"},
			{Local: "Id"},
		}
		require.Nil(t, findMask(otherId, masks))

		cdtr := []xml.Name{
			{Local: "CdtrAcct"},
			{Local: "Id"},
			{Local: "Othr"},
			{Local: "Id"},
		}
		require.Nil(t, findMask(cdtr, masks))
	})

	t.Run("longest path wins", func(t *testing.T) {
		masks := []Mask{
			{Name: "Id", Mask: ShowNone},
			{Name: "DbtrAcct/Id", Mask: ShowLastFour},
		}

		generic := []xml.Name{{Local: "Rpt"}, {Local: "Id"}}
		require.Equal(t, &masks[0], findMask(generic, masks))

		specific := []xml.Name{{Local: "DbtrAcct"}, {Local: "Id"}, {Local: "Othr"}, {Local: "Id"}}
		require.Equal(t, &masks[1], findMask(specific, masks))
	})

	t.Run("path with prefix", func(t *testing.T) {
		masks := []Mask{
			{Name: "ct:DbtrAcct/ct:Id", Mask: ShowLastFour},
		}

		matched := []xml.Name{
			{Space: "ct", Local: "DbtrAcct"},
			{Space: "ct", Local: "Id"},
		}
		require.Equal(t, &masks[0], findMask(matched, masks))

		wrongPrefix := []xml.Name{
			{Space: "mr", Local: "DbtrAcct"},
			{Space: "mr", Local: "Id"},
		}
		require.Nil(t, findMask(wrongPrefix, masks))
	})
}

func TestApplyMask(t *testing.T) {
	t.Run("last four", func(t *testing.T) {
		cases := []struct {
			input, expected string
		}{
			{"", ""},
			{" ", "*"},
			{"  ", "**"},
			{"123", "***"},
			{"1234", "****"},
			{"12345", "*2345"},
			{"123456", "**3456"},
			{"Adam Shannon", "********nnon"},
			{"José€", "*osé€"},
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
			{"1234", "*23*"},
			{"12345", "**3**"},
			{"123456", "**34**"},
			{"Adam Shannon", "**** Sha****"},
			{"José", "*os*"},
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
			{" ", " "},
			{"  ", "  "},
			{"123", "1**"},
			{"1 2 3", "1 2 3"},
			{"12 34 56", "1* 3* 5*"},
			{"123 456", "1** 4**"},
			{"Adam Shannon", "A*** S******"},
			{"  John   Doe  ", "  J***   D**  "},
			{"José María", "J*** M****"},
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
			{"1 2 3", "*****"},
			{"12 34 56", "********"},
			{"123 456", "*******"},
			{"Adam Shannon", "************"},
			{"José", "****"},
		}

		for i := range cases {
			output := applyMask(xml.CharData(cases[i].input), &Mask{Mask: ShowNone})
			require.Equal(t, cases[i].expected, string(output), fmt.Sprintf("input: %q", cases[i].input))
		}
	})

	t.Run("unknown type is a no-op", func(t *testing.T) {
		output := applyMask(xml.CharData("secret"), &Mask{Mask: "nope"})
		require.Equal(t, "secret", string(output))
	})

	t.Run("nil mask is a no-op", func(t *testing.T) {
		output := applyMask(xml.CharData("secret"), nil)
		require.Equal(t, "secret", string(output))
	})
}
