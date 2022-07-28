package registry

import "github.com/kubevela/kube-trigger/pkg/filter/builtin/cuevalidator"

func RegisterBuiltinFilters() error {
	// Register cue-validator
	cv := &cuevalidator.CUEValidator{}
	TypeRegistry.Register(cv.Type(), cv)

	return nil
}
