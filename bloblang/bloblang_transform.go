package bloblang

import (
	"context"

	"github.com/benthosdev/benthos/v4/public/bloblang"
	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
	"github.com/sudo-suhas/play-script-engine/structmap"
)

var mapping = `
asset.labels.script_engine = "bloblang"

map entity_name {
	root = this
	root.entity_name = match this.name {
		"ongoing_placed_and_waiting_acceptance_orders" => "customer_orders",
		"ongoing_orders" => "customer_orders",
		"merchant_avg_dispatch_arrival_time_10m" => "merchant_driver",
		"ongoing_accepted_orders" => "merchant_orders",
	}
}
asset.data.features = asset.data.features.map_each(f -> f.apply("entity_name"))

asset.owners = asset.owners.or([]).append({ "name": "Big Mom", "email": "big.mom@wholecakeisland.com" })

asset.url = urler(asset.name)

map urn_replace {
	root = this
	root.urn = this.urn.replace_all(".yonkou.io", "")
}

asset.lineage.upstreams = asset.lineage.upstreams.map_each(u -> if u.service == "kafka" {
	u.apply("urn_replace")
} else {
	u
})
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(_ context.Context, a *asset.Asset) error {
	const op = "bloblang.Transform"

	env := bloblang.NewEnvironment().
		WithDisabledImports().
		WithoutFunctions("env", "file", "hostname")

	if err := env.RegisterFunction("urler", func(args ...any) (bloblang.Function, error) {
		var name string
		if err := bloblang.NewArgSpec().
			StringVar(&name).
			Extract(args); err != nil {
			return nil, err
		}

		return func() (any, error) {
			return t.URLer(name), nil
		}, nil
	}); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("register function"), errors.WithErr(err))
	}

	exe, err := env.Parse(mapping)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("parse mapping"), errors.WithErr(err))
	}

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	m := structmap.Map(a)
	m["data"] = structmap.Map(data)
	var v interface{} = map[string]interface{}{"asset": m}
	if err := exe.Overlay(v, &v); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute mapping"), errors.WithErr(err))
	}

	if err := structmap.Struct(m["data"], &data); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("decode map"), errors.WithErr(err))
	}

	delete(m, "data")
	if err := structmap.Struct(m, a); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("decode map"), errors.WithErr(err))
	}

	a.Data, err = anypb.New(data)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
