package tengo

import (
	"context"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/sudo-suhas/xgo/errors"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
	"github.com/sudo-suhas/play-script-engine/structmap"
)

var script = []byte(`
text := import("text")

merge := func(m1, m2) {
	for k, v in m2 {
		m1[k] = v
	}
	return m1
}

asset.labels = merge({script_engine: "tengo"}, asset.labels)

for e in asset.data.entities {
	e.labels = merge({catch_phrase: "You talkin' to me?"}, e.labels)
}

for f in asset.data.features {
	if f.name == "ongoing_placed_and_waiting_acceptance_orders" || f.name == "ongoing_orders" {
		f.entity_name = "customer_orders"
	} else if f.name == "merchant_avg_dispatch_arrival_time_10m" {
		f.entity_name = "merchant_driver"
	} else if f.name == "ongoing_accepted_orders" {
		f.entity_name = "merchant_orders"
	}
}

asset.owners = append(asset.owners || [], { name: "Big Mom", email: "big.mom@wholecakeisland.com" })

asset.url = urler(asset.name)

for u in asset.lineage.upstreams {
	u.urn = u.service != "kafka" ? u.urn : text.replace(u.urn, ".yonkou.io", "", -1)
}
`)

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(ctx context.Context, a *asset.Asset) error {
	const op = "tengo.Transform"

	s := tengo.NewScript(script)
	s.SetImports(stdlib.GetModuleMap(
		"text", "rand", "times", "fmt", "base64", "json", "math", "enum",
	))

	wrapper, err := structmap.NewAssetWrapper(a)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	m, err := wrapper.EncodeWithoutTypes()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	for name, v := range map[string]interface{}{
		"asset": m,
		"urler": stdlib.FuncASRS(func(name string) string {
			return t.URLer(name)
		}),
	} {
		if err := s.Add(name, v); err != nil {
			return errors.E(errors.WithOp(op), errors.WithErr(err))
		}
	}

	res, err := s.RunContext(ctx)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute script"), errors.WithErr(err))
	}

	if err := wrapper.OverwriteWith(res.Get("asset").Map()); err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
