package jsonutil

import (
	"encoding/json"

	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func JsonMarshal(v any) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		opts := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}
		return opts.Marshal(m)
	}
	m, err := json.Marshal(v)
	return m, errs.Wrap(err)
}

func JsonUnmarshal(b []byte, v any) error {
	if m, ok := v.(proto.Message); ok {
		opts := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		return errs.Wrap(opts.Unmarshal(b, m))
	}
	return errs.Wrap(json.Unmarshal(b, v))
}
