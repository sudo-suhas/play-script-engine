package goja

import (
	"context"

	"github.com/dop251/goja"
	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

var script = `
asset.labels = Object.assign({ script_engine: 'goja' }, asset.labels);

for (const e of data.entities) {
	e.labels = Object.assign({ catch_phrase: 'Say hello to my little friend.' }, e.labels);
}

for (const f of data.features) {
	switch (f.name) {
		case 'ongoing_placed_and_waiting_acceptance_orders':
		case 'ongoing_orders':
			f.entity_name = 'customer_orders';
			break;
		case 'merchant_avg_dispatch_arrival_time_10m':
			f.entity_name = 'merchant_driver';
			break;
		case 'ongoing_accepted_orders':
			f.entity_name = 'merchant_orders';
			break;
	}
}

asset.owners = [{ name: 'Big Mom', email: 'big.mom@wholecakeisland.com' }].concat(asset.owners);

asset.url = urler(asset.name);

for (const u of asset.lineage.upstreams) {
	if (u.service !== 'kafka') continue;
	
	u.urn = u.urn.replace('.yonkou.io', '');
}
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(_ context.Context, a *asset.Asset) error {
	const op = "goja.Transform"

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	for name, v := range map[string]interface{}{
		"asset": a,
		"data":  data,
		"urler": func(call goja.FunctionCall) goja.Value {
			url := t.URLer(call.Argument(0).String())
			return vm.ToValue(url)
		},
	} {
		if err := vm.Set(name, v); err != nil {
			return errors.E(errors.WithOp(op), errors.WithErr(err))
		}
	}

	if _, err := vm.RunString(script); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute js script"), errors.WithErr(err))
	}

	a.Data, err = anypb.New(data)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
