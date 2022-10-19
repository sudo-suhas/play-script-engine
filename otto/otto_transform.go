package otto

import (
	"context"

	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore" // add _ helpers to JS env
	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

var script = `
asset.labels = _.extend({ script_engine: 'otto' }, asset.labels);

_.each(data.entities, function(e) {
	e.labels = _.extend({ catch_phrase: 'I\'ll be back' }, e.labels);
});

_.each(data.features, function(f) {
	switch (f.name) {
		case 'ongoing_placed_and_waiting_acceptance_orders':
		case 'ongoing_orders':
			f.EntityName = 'customer_orders';
			break;
		case 'merchant_avg_dispatch_arrival_time_10m':
			f.EntityName = 'merchant_driver';
			break;
		case 'ongoing_accepted_orders':
			f.EntityName = 'merchant_orders';
			break;
	}
})

asset.owners = [{ name: 'Big Mom', email: 'big.mom@wholecakeisland.com' }].concat(asset.owners);

asset.url = urler(asset.name);

_.chain(asset.lineage.upstreams)
	.filter(function(u) { return u.service === 'kafka'; })
	.each(function(u) { u.urn = u.urn.replace('.yonkou.io', ''); });
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(_ context.Context, a *asset.Asset) (err error) {
	const op = "otto.Transform"

	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = errors.E(errors.WithOp(op), errors.WithText("panic recovered"), errors.WithErr(e))
				return
			}
			err = errors.E(errors.WithOp(op), errors.WithTextf("%v", r), errors.WithData(r))
		}
	}()

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	vm := otto.New()
	for name, v := range map[string]interface{}{
		"asset": a,
		"data":  data,
		"urler": func(call otto.FunctionCall) otto.Value {
			v, _ := vm.ToValue(t.URLer(call.Argument(0).String()))
			return v
		},
	} {
		if err := vm.Set(name, v); err != nil {
			return errors.E(errors.WithOp(op), errors.WithErr(err))
		}
	}

	if _, err := vm.Run(script); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute js script"), errors.WithErr(err))
	}

	a.Data, err = anypb.New(data)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
