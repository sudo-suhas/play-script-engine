package gopherlua

import (
	"context"

	"github.com/spy16/pkg/lua"
	"github.com/sudo-suhas/xgo/errors"
	luastd "github.com/yuin/gopher-lua"
	"google.golang.org/protobuf/types/known/anypb"
	luar "layeh.com/gopher-luar"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

var script = `
if asset.labels == nil then
	asset.labels = {}
end
asset.labels["script_engine"] = "gopherlua"

for _, e in data.entities() do
	if e.labels == nil then
		e.labels = {}
	end
	e.labels["catch_phrase"] = "You Shall Not Pass!"
end

for _, f in data.features() do
	if f.name == "ongoing_placed_and_waiting_acceptance_orders" or f.name == "ongoing_orders" then
		f.entityName = "customer_orders"
	elseif f.name == "merchant_avg_dispatch_arrival_time_10m" then
		f.entityName = "merchant_driver"
	elseif f.name == "ongoing_accepted_orders" then
		f.entityName = "merchant_orders"
	end
end

if asset.owners == nil then
	asset.owners = {}
end
asset.owners = asset.owners + {Name = "Big Mom", Email = "big.mom@wholecakeisland.com"}

asset.url = urler(asset.name)

for _, u in asset.lineage.upstreams() do
	if u.service == "kafka" then
		u.urn = u.urn:gsub("\.yonkou\.io", "") 
	end
end
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(ctx context.Context, a *asset.Asset) error {
	const op = "gopherlua.Transform"

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	l, err := lua.New(
		lua.Context(ctx),
		lua.Globals(map[string]interface{}{
			"asset": a,
			"data":  data,
			"urler": func(L *luar.LState) int {
				u := t.URLer(L.CheckString(1))
				L.Push(luastd.LString(u))
				return 1
			},
		}),
	)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("init new lua state"), errors.WithErr(err))
	}

	if err := l.Execute(script); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute lua script"), errors.WithErr(err))
	}

	a.Data, err = anypb.New(data)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
