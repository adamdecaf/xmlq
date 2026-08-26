# xmlq

[![GoDoc](https://godoc.org/github.com/adamdecaf/xmlq?status.svg)](https://pkg.go.dev/github.com/adamdecaf/xmlq/pkg/xmlq)
[![Build Status](https://github.com/adamdecaf/xmlq/workflows/Go/badge.svg)](https://github.com/adamdecaf/xmlq/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/adamdecaf/xmlq)](https://goreportcard.com/report/github.com/adamdecaf/xmlq)
[![Apache 2 License](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/adamdecaf/xmlq/master/LICENSE)

xmlq is a Go library for pretty printing xml and masking element values. Masks can target a single element name (`Id`) or a partial ancestor chain (`DbtrAcct/Id`) so identifiers you need to keep, such as `<Rpt><Id>`, are left alone. XML inside of CDATA stanzas is also masked and indented.

## Install

Download the [latest release for your architecture](https://github.com/adamdecaf/xmlq/releases/latest).

You can install from source:
```
go install github.com/adamdecaf/xmlq/cmd/xmlq@latest
```

## Usage

### Command line

Read files
```
xmlq ./pkg/xmlq/testdata/admi_002.xml
```
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Message xmlns="urn:tch" xmlns:_xmlns="xmlns" _xmlns:head="urn:iso:std:iso:20022:tech:xsd:head.001.001.01">
  <AppHdr>
  ...
  </AppHdr>
  <MessageReject>
  ...
  </MessageReject>
</Message>
```

Read from stdin
```
cat ./pkg/xmlq/testdata/admi_002.xml | xmlq
```

### Go library

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
			// <DbtrAcct><Id>…</Id></DbtrAcct> — including nested
			// <DbtrAcct><Id><Othr><Id>. Does not match <Rpt><Id>.
			Name: "DbtrAcct/Id",
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
