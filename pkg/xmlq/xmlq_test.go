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
		marshal(t, filepath.Join("testdata", "note.xml"), filepath.Join("testdata", "note.expected.xml"))
	})
	t.Run("pacs_008.xml", func(t *testing.T) {
		marshal(t, filepath.Join("testdata", "pacs_008.xml"), filepath.Join("testdata", "pacs_008.xml"))
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
			{
				Name: "from",
				Mask: ShowMiddle,
			},
		},
	})
	require.NoError(t, err)

	bs, err := os.ReadFile(expected)
	require.NoError(t, err)

	fmt.Printf("\n---\n%s\n---\n", string(output))

	require.Equal(t, string(bs), string(output))
}
