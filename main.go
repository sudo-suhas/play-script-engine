package main

import (
	"context"
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/sudo-suhas/xgo/errors"
	"github.com/sudo-suhas/xgo/httputil"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/sudo-suhas/play-script-engine/goja"
	"github.com/sudo-suhas/play-script-engine/gopherlua"
	"github.com/sudo-suhas/play-script-engine/otto"
	"github.com/sudo-suhas/play-script-engine/proto/asset"
	"github.com/sudo-suhas/play-script-engine/sample"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lg := log.New()
	lg.SetOutput(os.Stdout)
	lg.SetFormatter(&log.JSONFormatter{
		DisableHTMLEscape: true,
		PrettyPrint:       true,
	})

	logger := lg.WithField("source", "main")
	if err := run(ctx, os.Args[1:], logger); err != nil {
		logger.WithError(err).Fatalln("run failed")
	}
}

func run(ctx context.Context, args []string, logger log.FieldLogger) error {
	const op = "run"

	engine := "goja"
	if len(args) != 0 {
		engine = args[0]
	}

	a, err := sample.FeatureTable()
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	ub, err := httputil.NewURLBuilderSource("https://my-dummy-domain.company.com/")
	if err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	urler := func(name string) string { return ub.NewURLBuilder().Path(name).URL().String() }

	var t transformer
	switch engine {
	case "gopherlua":
		t = &gopherlua.Transformer{URLer: urler}

	case "otto":
		t = &otto.Transformer{URLer: urler}

	case "goja":
		t = &goja.Transformer{URLer: urler}

	default:
		return errors.E(errors.WithOp(op), errors.WithTextf("unknown script engine: %s", engine))
	}

	if err := t.T(ctx, a); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("transform"), errors.WithErr(err))
	}

	data, _ := protojson.Marshal(a.Data)
	logger.WithField("data", json.RawMessage(data)).
		WithField("asset", a).
		Info("Transformed")

	return nil
}

type transformer interface {
	// T should do the following:
	// - Add a label to the asset - "script_engine": "<current_script_engine>
	// - Add a label to each entity. Ex: "catch_phrase": "..."
	// - Set an EntityName for each feature based on the following table
	//   - ongoing_placed_and_waiting_acceptance_orders: customer_orders
	//   - ongoing_orders: customer_orders
	//   - merchant_avg_dispatch_arrival_time_10m: merchant_driver
	//   - ongoing_accepted_orders: merchant_orders
	// - Set the owner as {Name: Big Mom, Email: big.mom@wholecakeisland.com}
	// - Set the Url using a function that is passed in
	// - For each lineage upstream, if the service is Kafka, apply a string
	//   replace on the URN - {.yonkou.io => }.
	T(ctx context.Context, a *asset.Asset) error
}
