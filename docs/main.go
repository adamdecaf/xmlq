package main

import (
	"bytes"
	"syscall/js"

	"github.com/adamdecaf/xmlq/pkg/xmlq"
)

func maskXML(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return map[string]interface{}{
			"error": "Expected input XML and options",
		}
	}

	input := args[0].String()
	optsObj := args[1]

	// Parse options from JavaScript object
	opts := &xmlq.Options{
		Prefix: optsObj.Get("prefix").String(),
		Indent: optsObj.Get("indent").String(),
		Masks:  []xmlq.Mask{},
	}

	// Read masks from JavaScript array
	masks := optsObj.Get("masks")
	for i := 0; i < masks.Length(); i++ {
		mask := masks.Index(i)
		opts.Masks = append(opts.Masks, xmlq.Mask{
			Name:  mask.Get("name").String(),
			Space: mask.Get("space").String(),
			Mask:  xmlq.MaskingType(mask.Get("mask").String()),
		})
	}

	// Process XML
	result, err := xmlq.MarshalIndent(bytes.NewReader([]byte(input)), opts)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"result": string(result),
	}
}

func main() {
	js.Global().Set("maskXML", js.FuncOf(maskXML))
	select {} // Keep the program running
}
