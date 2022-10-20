package structmap

import (
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/sudo-suhas/xgo/errors"
)

func AsMap(v interface{}) map[string]interface{} {
	s := structs.New(v)
	s.TagName = "json"
	return s.Map()
}

func AsStruct(input, output interface{}) error {
	const op = "structmap.AsStruct"

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		ZeroFields:  true,
		Result:      output,
		TagName:     "json",
	})
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("mapstructure config"), errors.WithErr(err))
	}

	if err := dec.Decode(input); err != nil {
		return errors.E(errors.WithOp(op), errors.WithTextf("decode into %T", output), errors.WithErr(err))
	}

	return nil
}
