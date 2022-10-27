package golua

import (
	"context"

	"github.com/Shopify/go-lua"
	luautil "github.com/Shopify/goluago/util"
	"github.com/sudo-suhas/xgo/errors"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
	"github.com/sudo-suhas/play-script-engine/structmap"
)

var script = `
if asset.labels == nil then
	asset.labels = {}
end
asset.labels["script_engine"] = "gopherlua"

for _, e in ipairs(asset.data.entities) do
	if e.labels == nil then
		e.labels = {}
	end
	e.labels["catch_phrase"] = "Here’s Johnny!"
end

for _, f in ipairs(asset.data.features) do
	if f.name == "ongoing_placed_and_waiting_acceptance_orders" or f.name == "ongoing_orders" then
		f.entity_name = "customer_orders"
	elseif f.name == "merchant_avg_dispatch_arrival_time_10m" then
		f.entity_name = "merchant_driver"
	elseif f.name == "ongoing_accepted_orders" then
		f.entity_name = "merchant_orders"
	end
end

if asset.owners == nil then
	-- asset.owners gets initialised as a map and decoding into *asset.Asset fails.
	-- asset.owners = {}
end
-- Inserting into the table fails with runtime error: invalid key to 'next'
-- table.insert(asset.owners, {name = "Big Mom", email = "big.mom@wholecakeisland.com"})

asset.url = urler(asset.name)

for _, u in ipairs(asset.lineage.upstreams) do
	if u.service == "kafka" then
		-- Fails inexplicably with "attempt to call a nil value" runtime error.
		-- u.urn = u.urn:gsub(".yonkou.io", "") 
	end
end
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(_ context.Context, a *asset.Asset) error {
	const op = "golua.Transform"

	l := lua.NewState()
	libs := []lua.RegistryFunction{
		{Name: "_G", Function: lua.BaseOpen},
		{Name: "package", Function: lua.PackageOpen},
		{Name: "table", Function: lua.TableOpen},
		{Name: "string", Function: lua.StringOpen},
		{Name: "math", Function: lua.MathOpen},
	}
	for _, lib := range libs {
		lua.Require(l, lib.Name, lib.Function, true)
	}

	wrapper, err := structmap.NewAssetWrapper(a)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	luautil.DeepPush(l, wrapper.Encode())
	l.SetGlobal("asset")

	l.Register("urler", func(l *lua.State) int {
		u := t.URLer(lua.CheckString(l, 1))
		l.PushString(u)
		return 1
	})

	var (
		v       interface{}
		pullErr error
	)
	l.Register("pull_table", func(l *lua.State) int {
		v, pullErr = luautil.PullTable(l, 1)
		return 0
	})

	if err := lua.DoString(l, script+"\npull_table(asset)"); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute lua script"), errors.WithErr(err))
	}
	if pullErr != nil {
		return errors.E(errors.WithOp(op), errors.WithText("execute lua script"), errors.WithErr(err))
	}

	res, ok := v.(map[string]interface{})
	if !ok {
		return errors.E(errors.WithOp(op), errors.WithTextf("unexpected result: %T", v), errors.WithErr(err))
	}

	if err := wrapper.OverwriteWith(res); err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return nil
}
