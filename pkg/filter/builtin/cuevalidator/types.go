package cuevalidator

import "cuelang.org/go/cue"

const (
	TemplateFieldName = "template"
)

type Properties struct {
	Template cue.Value
}
