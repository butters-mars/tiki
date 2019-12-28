package http

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// SettingProvider provides endpoint settings for a given uri target.
type SettingProvider interface {
	GetSettings(target string) (map[string]EndpointSetting, error)
	SetHandler(handler func(setting EndpointSetting) error)
}

type settingHolder struct {
	Settings map[string][]EndpointSetting
}

// FileSettingProvider implements a file-based provider.
type FileSettingProvider struct {
	path       string
	settingMap map[string]map[string]EndpointSetting
}

// NewFileSettingProvider creates a new setting provider with given yaml file.
func NewFileSettingProvider(path string) (*FileSettingProvider, error) {
	p := &FileSettingProvider{
		path: path,
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Error("fail to load setting from %s", path)
		return nil, err
	}

	holder := &settingHolder{}
	err = yaml.Unmarshal(data, holder)
	if err != nil {
		return nil, err
	}

	settingMap := make(map[string]map[string]EndpointSetting)
	for target, settingsByTarget := range holder.Settings {
		epMap := make(map[string]EndpointSetting)
		for _, setting := range settingsByTarget {
			key := fmt.Sprintf("%s-%s", setting.Method, setting.URI)
			epMap[key] = setting
		}

		settingMap[target] = epMap
	}

	p.settingMap = settingMap

	return p, nil

}

// GetSettings implements SettingProvider.GetSettings
func (p FileSettingProvider) GetSettings(target string) (map[string]EndpointSetting, error) {
	epMap, ok := p.settingMap[target]
	if !ok {
		epMap = make(map[string]EndpointSetting)
	}

	return epMap, nil
}

// SetHandler implements SettingProvider.SetHandler
func (p FileSettingProvider) SetHandler(handler func(setting EndpointSetting) error) {

}
