package cuevalidator

type Properties struct {
	CUE cueTmpl `json:"cue"`
}

type cueTmpl struct {
	Template template `json:"template"`
}

type template string

func (t template) String() string {
	return string(t)
}
