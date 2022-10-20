package gojq

import (
	"context"

	"github.com/itchyny/gojq"
	"github.com/sudo-suhas/xgo/errors"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
	"github.com/sudo-suhas/play-script-engine/structmap"
)

var script = `
.labels.script_engine = "gojq" | 

.data.entities[].labels.catch_phrase = "Go ahead. Make my day." |

.data.features[] |= 
    if .name == "ongoing_placed_and_waiting_acceptance_orders" or .name == "ongoing_orders" then 
		.entity_name = "customer_orders" 
    elif .name == "merchant_avg_dispatch_arrival_time_10m" then 
		.entity_name = "merchant_driver"
    elif .name == "ongoing_accepted_orders" then 
		.entity_name = "merchant_orders"
	else . end |

.owners += [{name: "Big Mom", email: "big.mom@wholecakeisland.com"}] |

.url = urler(.name) |

.lineage.upstreams[] |=
    if .service == "kafka" then .urn = (.urn | sub("\\.yonkou\\.io"; ""))  
    else . end
`

type Transformer struct {
	URLer func(string) string
}

func (t *Transformer) T(ctx context.Context, a *asset.Asset) error {
	const op = "gojq.Transform"

	query, err := gojq.Parse(script)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("parse query"), errors.WithErr(err))
	}

	code, err := gojq.Compile(
		query,
		gojq.WithFunction(
			"urler", 1, 1,
			func(jqCtx interface{}, args []interface{}) interface{} {
				s, ok := args[0].(string)
				if !ok {
					return errors.E(errors.WithOp("gojq.urler"), errors.WithTextf("unexpected type: %T", s))
				}

				return t.URLer(s)
			},
		),
	)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("compile query"), errors.WithErr(err))
	}

	wrapper, err := structmap.NewAssetWrapper(a)
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	m, err := wrapper.EncodeWithoutTypes()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	iter := code.RunWithContext(ctx, m)
	v, ok := iter.Next()
	if !ok {
		return errors.E(errors.WithOp(op), errors.WithText("unexpected result"), errors.WithErr(err))
	}

	switch v := v.(type) {
	case error:
		return errors.E(errors.WithOp(op), errors.WithText("run query"), errors.WithErr(err))

	case map[string]interface{}:
		if err := wrapper.OverwriteWith(v); err != nil {
			return errors.E(errors.WithOp(op), errors.WithErr(err))
		}

	default:
		return errors.E(errors.WithOp(op), errors.WithTextf("unexpected result: %T", v), errors.WithErr(err))
	}

	return nil
}
