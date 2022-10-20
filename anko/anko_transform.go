package anko

import (
	"context"
	"fmt"

	"github.com/mattn/anko/env"
	_ "github.com/mattn/anko/packages" // Protect Me, O My Lord
	"github.com/mattn/anko/vm"
	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

var script = `
strings = import("strings")

func merge(m1, m2) {
	for k, v in m2 {
		m1[k] = v
	}
	return m1
}

asset.Labels = merge({"script_engine": "anko"}, asset.Labels)

for e in data.Entities {
	e.Labels = merge({"catch_phrase": "Take your stinking paws off me, you damn dirty ape!"}, e.Labels)
}

for f in data.Features {
	if f.Name == "ongoing_placed_and_waiting_acceptance_orders" || f.Name == "ongoing_orders" {
		f.EntityName = "customer_orders"
	} else if f.Name == "merchant_avg_dispatch_arrival_time_10m" {
		f.EntityName = "merchant_driver"
	} else if f.Name == "ongoing_accepted_orders" {
		f.EntityName = "merchant_orders"
	}
}

o = make(Owner)
o.Name = "Big Mom"
o.Email = "big.mom@wholecakeisland.com"
asset.Owners += o

asset.Url = urler(asset.Name)

for u in asset.Lineage.Upstreams {
	if u.Urn != "kafka" {
		continue
	}
	u.Urn = strings.Replace(u.Urn, ".yonkou.io", "", -1)
}
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(ctx context.Context, a *asset.Asset) error {
	const op = "anko.Transform"

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	e := env.NewEnv()
	for name, v := range map[string]interface{}{
		"asset":   a,
		"data":    data,
		"println": fmt.Println,
		"urler":   t.URLer,
	} {
		if err := e.DefineGlobal(name, v); err != nil {
			return errors.E(errors.WithOp(op), errors.WithErr(err))
		}
	}
	if err := e.DefineType("Owner", &asset.Owner{}); err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}
	if _, err := vm.ExecuteContext(ctx, e, nil, script); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute script"), errors.WithErr(err))
	}

	a.Data, err = anypb.New(data)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
