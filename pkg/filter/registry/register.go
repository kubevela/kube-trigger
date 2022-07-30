package registry

import (
	"github.com/kubevela/kube-trigger/pkg/filter/builtin/cuevalidator"
	"github.com/kubevela/kube-trigger/pkg/filter/types"
)

func RegisterBuiltinFilters(reg *Registry) {

	// Register cue-validator
	cv := &cuevalidator.CUEValidator{}
	cvMeta := types.FilterMeta{
		Type: cv.Type(),
	}
	reg.RegisterType(cvMeta, cv)

}
