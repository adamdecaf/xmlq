package xmlq

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalIndent(t *testing.T) {
	t.Run("note.xml", func(t *testing.T) {
		output := marshal(t, filepath.Join("testdata", "note.xml"))
		fmt.Printf("\n---\n%s\n---\n", string(output))
	})

	t.Run("pacs_008.xml", func(t *testing.T) {
		output := marshal(t, filepath.Join("testdata", "pacs_008.xml"))
		fmt.Printf("\n---\n%s\n---\n", string(output))
	})
}

func marshal(t *testing.T, path string) []byte {
	t.Helper()

	fd, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { fd.Close() })

	output, err := MarshalIndent(fd, &Options{
		Indent: "  ",
		Masks: []Mask{
			{
				Name: "from",
				Mask: ShowMiddle,
			},
		},
	})
	require.NoError(t, err)

	return output
}
