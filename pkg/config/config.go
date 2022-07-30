package config

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	actiontype "github.com/kubevela/kube-trigger/pkg/action/types"
	filtertype "github.com/kubevela/kube-trigger/pkg/filter/types"
	sourcetype "github.com/kubevela/kube-trigger/pkg/source/types"
	utilcue "github.com/kubevela/kube-trigger/pkg/utils/cue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Watch   []WatchMeta
	Actions []actiontype.ActionMeta
}

type WatchMeta struct {
	Source  sourcetype.SourceMeta
	Filters []filtertype.FilterMeta
}

const (
	WatchFieldName   = "watch"
	SourceFieldName  = "source"
	FiltersFieldName = "filters"
	ActionsFieldName = "actions"
)

func (c *Config) Parse(confStr string) error {
	var err error

	confStr = "context: _\n" + confStr

	cueCtx := cuecontext.New()
	vConf := cueCtx.CompileString(confStr)

	err = vConf.Validate()
	if err != nil {
		return err
	}

	vWatches := vConf.LookupPath(cue.ParsePath(WatchFieldName))
	if vWatches.Err() != nil {
		return vWatches.Err()
	}

	c.Watch, err = parseWatches(vWatches)
	if err != nil {
		return err
	}

	vActions := vConf.LookupPath(cue.ParsePath(ActionsFieldName))
	if vActions.Err() != nil {
		return vActions.Err()
	}

	c.Actions, err = parseActions(vActions)
	if err != nil {
		return err
	}

	return nil
}

func parseWatches(vWatches cue.Value) ([]WatchMeta, error) {
	var ret []WatchMeta

	vWatchList, err := vWatches.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vWatchList.Next(); i++ {
		watch, err := parseWatch(vWatchList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", WatchFieldName, i)
		}
		ret = append(ret, watch)
	}

	return ret, nil
}

func parseWatch(vWatch cue.Value) (WatchMeta, error) {
	var err error
	ret := WatchMeta{}

	vSource := vWatch.LookupPath(cue.ParsePath(SourceFieldName))
	if vSource.Err() != nil {
		return ret, vSource.Err()
	}

	ret.Source, err = parseSource(vSource)
	if err != nil {
		return ret, err
	}

	vFilters := vWatch.LookupPath(cue.ParsePath(FiltersFieldName))
	if vFilters.Err() != nil {
		return ret, vFilters.Err()
	}

	ret.Filters, err = parseFilters(vFilters)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func parseSource(vSource cue.Value) (sourcetype.SourceMeta, error) {
	var err error
	ret := sourcetype.SourceMeta{}

	vType := vSource.LookupPath(cue.ParsePath(sourcetype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vSource.LookupPath(cue.ParsePath(sourcetype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	ret.Properties = vProperties

	logrus.WithField("config", "parser").Debugf("parsed source: %s", ret.Type)

	return ret, nil
}

func parseFilters(vFilters cue.Value) ([]filtertype.FilterMeta, error) {
	var ret []filtertype.FilterMeta

	vFilterList, err := vFilters.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vFilterList.Next(); i++ {
		filter, err := parseFilter(vFilterList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", FiltersFieldName, i)
		}
		ret = append(ret, filter)
	}

	return ret, nil
}

func parseFilter(vFilter cue.Value) (filtertype.FilterMeta, error) {
	var err error
	ret := filtertype.FilterMeta{}

	vType := vFilter.LookupPath(cue.ParsePath(filtertype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vFilter.LookupPath(cue.ParsePath(filtertype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	rawStr, err := utilcue.Marshal(vFilter)
	if err != nil {
		return ret, err
	}
	ret.Raw = rawStr
	ret.Properties = vProperties

	logrus.WithField("config", "parser").Debugf("parsed filter: %s", ret.Raw)

	return ret, nil
}

func parseActions(vActions cue.Value) ([]actiontype.ActionMeta, error) {
	var ret []actiontype.ActionMeta

	vActionsList, err := vActions.List()
	if err != nil {
		return nil, err
	}
	for i := 0; vActionsList.Next(); i++ {
		action, err := parseAction(vActionsList.Value())
		if err != nil {
			return nil, errors.Wrapf(err, "error when parsing %s[%d]", ActionsFieldName, i)
		}
		ret = append(ret, action)
	}

	return ret, nil
}

func parseAction(vAction cue.Value) (actiontype.ActionMeta, error) {
	var err error
	ret := actiontype.ActionMeta{}

	vType := vAction.LookupPath(cue.ParsePath(actiontype.TypeFieldName))
	if vType.Err() != nil {
		return ret, vType.Err()
	}
	vProperties := vAction.LookupPath(cue.ParsePath(actiontype.PropertiesFieldName))
	if vProperties.Err() != nil {
		return ret, vProperties.Err()
	}

	ret.Type, err = vType.String()
	if err != nil {
		return ret, err
	}
	rawStr, err := utilcue.Marshal(vAction)
	if err != nil {
		return ret, err
	}
	ret.Raw = rawStr
	ret.Properties = vProperties

	logrus.WithField("config", "parser").Debugf("parsed action: %s", ret.Raw)

	return ret, nil
}
