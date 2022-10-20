package structmap

import (
	"encoding/json"

	"github.com/sudo-suhas/xgo/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sudo-suhas/play-script-engine/proto/asset"
)

type AssetWrapper struct {
	*asset.Asset
	UnmarshaledData proto.Message
}

func NewAssetWrapper(a *asset.Asset) (*AssetWrapper, error) {
	const op = "structmap.NewAssetWrapper"

	data, err := a.Data.UnmarshalNew()
	if err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return &AssetWrapper{
		Asset:           a,
		UnmarshaledData: data,
	}, nil
}

func (w *AssetWrapper) Encode() map[string]interface{} {
	m := AsMap(w.Asset)
	m["data"] = AsMap(w.UnmarshaledData)
	return m
}

func (w *AssetWrapper) EncodeWithoutTypes() (map[string]interface{}, error) {
	const op = "assetWrapper.EncodeWithoutTypes"

	m, err := mapWithoutTypes(w.Asset)
	if err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	m["data"], err = mapWithoutTypes(w.UnmarshaledData)
	if err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return m, nil
}

func (w *AssetWrapper) OverwriteWith(m map[string]interface{}) error {
	const op = "assetWrapper.OverwriteWith"

	if err := AsStruct(m["data"], &w.UnmarshaledData); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("decode data map"), errors.WithErr(err))
	}

	delete(m, "data")
	if err := AsStruct(m, w.Asset); err != nil {
		return errors.E(errors.WithOp(op), errors.WithText("decode asset map"), errors.WithErr(err))
	}

	var err error
	if w.Data, err = anypb.New(w.UnmarshaledData); err != nil {
		return errors.E(errors.WithOp(op), errors.WithErr(err))
	}
	return nil
}

func mapWithoutTypes(v interface{}) (map[string]interface{}, error) {
	const op = "tengo.map"

	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errors.E(errors.WithOp(op), errors.WithErr(err))
	}

	return res, nil
}
