package cue

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"k8s.io/apimachinery/pkg/util/json"
)

func Marshal(v cue.Value) (string, error) {
	syn := v.Syntax(cue.Raw())
	bs, err := format.Node(syn)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// Must be a pointer
func UnMarshal(v cue.Value, dst map[string]interface{}) error {
	jsonByte, err := v.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonByte, &dst)
	if err != nil {
		return err
	}
	return nil
}
