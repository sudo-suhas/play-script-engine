package sample

import (
	"time"

	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

func FeatureTable() (*asset.Asset, error) {
	const op = "sample.FeatureTable"

	featureTable, err := anypb.New(&asset.FeatureTable{
		Namespace: "sauron",
		Entities: []*asset.FeatureTable_Entity{
			{Name: "merchant_uuid", Labels: map[string]string{
				"description": "merchant uuid",
				"value_type":  "STRING",
			}},
		},
		Features: []*asset.Feature{
			{Name: "ongoing_placed_and_waiting_acceptance_orders", DataType: "INT64"},
			{Name: "ongoing_orders", DataType: "INT64"},
			{Name: "merchant_avg_dispatch_arrival_time_10m", DataType: "FLOAT"},
			{Name: "ongoing_accepted_orders", DataType: "INT64"},
		},
		CreateTime: timestamppb.New(time.Date(2022, time.September, 19, 22, 42, 0o4, 0, time.UTC)),
		UpdateTime: timestamppb.New(time.Date(2022, time.September, 21, 13, 23, 0o2, 0, time.UTC)),
	})
	if err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithText("feature table data"), errors.WithErr(err))
	}

	return &asset.Asset{
		Urn:     "urn:caramlstore:test-caramlstore:feature_table:avg_dispatch_arrival_time_10_mins",
		Name:    "avg_dispatch_arrival_time_10_mins",
		Service: "caramlstore",
		Type:    "feature_table",
		Data:    featureTable,
		Lineage: &asset.Lineage{
			Upstreams: []*asset.Resource{
				{
					Urn:     "urn:kafka:int-dagstream-kafka.yonkou.io:topic:GO_FOOD-delay-allocation-merchant-feature-10m-log",
					Service: "kafka",
					Type:    "topic",
				},
			},
		},
	}, nil
}
