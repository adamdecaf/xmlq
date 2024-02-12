# xmlq

[![GoDoc](https://godoc.org/github.com/adamdecaf/xmlq?status.svg)](https://pkg.go.dev/github.com/adamdecaf/xmlq/pkg/xmlq)
[![Build Status](https://github.com/adamdecaf/xmlq/workflows/Go/badge.svg)](https://github.com/adamdecaf/xmlq/actions)
[![Coverage Status](https://codecov.io/gh/adamdecaf/xmlq/branch/master/graph/badge.svg)](https://codecov.io/gh/adamdecaf/xmlq)
[![Go Report Card](https://goreportcard.com/badge/github.com/adamdecaf/xmlq)](https://goreportcard.com/report/github.com/adamdecaf/xmlq)
[![Apache 2 License](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/adamdecaf/xmlq/master/LICENSE)

xmlq is a Go library for pretty printing xml and masking element values.

## Usage

```go
import (
	"github.com/adamdecaf/xmlq/pkg/xmlq"
)

var (
	xmlData io.Reader
)

output, err := xmlq.MarshalIndent(xmlData, &Options{
	Indent: "  ", // two spaces
	Masks: []Mask{
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
	},
})
```

## Supported and tested platforms

- 64-bit Linux (Ubuntu, Debian), macOS, and Windows

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
