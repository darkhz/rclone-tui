package rclone

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// ConfigProviders stores the list of configuration providers.
type ConfigProviders struct {
	Providers []Provider `json:"providers"`
}

// Provider stores the provider information.
type Provider struct {
	Name        string           `json:"Name"`
	Description string           `json:"Description"`
	Prefix      string           `json:"Prefix"`
	Options     []ProviderOption `json:"Options"`
}

// ProviderOption stores the provider's option data.
type ProviderOption struct {
	Name       string `json:"Name"`
	Help       string `json:"Help"`
	Provider   string `json:"Provider"`
	ShortOpt   string `json:"ShortOpt"`
	Hide       int    `json:"Hide"`
	Required   bool   `json:"Required"`
	IsPassword bool   `json:"IsPassword"`
	NoPrefix   bool   `json:"NoPrefix"`
	Advanced   bool   `json:"Advanced"`
	Exclusive  bool   `json:"Exclusive"`
	DefaultStr string `json:"DefaultStr"`
	ValueStr   string `json:"ValueStr"`
	Type       string `json:"Type"`
	Examples   []struct {
		Value    string `json:"Value"`
		Help     string `json:"Help"`
		Provider string `json:"Provider"`
	} `json:"Examples"`
}

var (
	store           ConfigProviders
	currentSettings map[string]map[string]interface{}
)

// CacheConfigProviders fetches the provider list and stores it.
func CacheConfigProviders() error {
	response, err := SendCommand(map[string]interface{}{}, "/config/providers")
	if err != nil {
		return err
	}

	return response.Decode(&store)
}

// GetConfigProviders returns the list of providers.
func GetConfigProviders() (ConfigProviders, error) {
	var err error

	if store.Providers != nil {
		goto CachedProvider
	}

	err = CacheConfigProviders()
	if err != nil {
		return ConfigProviders{}, err
	}

CachedProvider:
	return store, nil
}

// GetConfigSettings returns the list of configured remotes.
func GetConfigSettings() (map[string]map[string]interface{}, error) {
	var settings map[string]map[string]interface{}

	job, err := SendCommandAsync("UI:Configuration", "Getting configuration", map[string]interface{}{}, "/config/dump")
	if err != nil {
		return nil, err
	}

	jobInfo, err := GetJobReply(job)
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(jobInfo.Output, &settings)

	if settings != nil {
		currentSettings = settings
	}

	return settings, err
}

// GetCurrentSettings returns the cached list of remotes.
func GetCurrentSettings() map[string]map[string]interface{} {
	settings := make(map[string]map[string]interface{})

	for name, setting := range currentSettings {
		if settings[name] == nil {
			settings[name] = make(map[string]interface{})
		}

		settings[name] = setting
	}

	return settings
}

// GetProviderByDesc matches and returns a provider by its description.
func GetProviderByDesc(confDesc string) (Provider, error) {
	providers, err := GetConfigProviders()
	if err != nil {
		return Provider{}, err
	}

	for _, p := range providers.Providers {
		if strings.Index(p.Description, confDesc) != -1 {
			return p, nil
		}
	}

	return Provider{}, fmt.Errorf("Could not find provider")
}

// GetProviderByType matches and returns a provider by its type.
func GetProviderByType(confType string) (Provider, error) {
	providers, err := GetConfigProviders()
	if err != nil {
		return Provider{}, err
	}

	for _, p := range providers.Providers {
		if p.Prefix == confType {
			return p, nil
		}
	}

	return Provider{}, fmt.Errorf("Could not find provider")
}

// GetProviderDesc returns the providers description.
func GetProviderDesc(confType string) string {
	var confDesc string

	providers, err := GetConfigProviders()
	if err != nil {
		return ""
	}

	for _, provider := range providers.Providers {
		if provider.Prefix == confType {
			confDesc = provider.Description
			break
		}
	}

	return confDesc
}

// SaveConfig saves the configuration.
func SaveConfig(data map[string]interface{}, create, interactiveConfig bool) error {
	mode := "create"
	if !create {
		mode = "update"
	}

	command := make(map[string]interface{})
	name := data["name"]

	delete(data, "name")
	delete(data, "configuration")
	if !create {
		delete(data, "type")
	} else {
		command["type"] = data["type"]
	}

	command["name"] = name
	command["parameters"] = data
	command["opt"] = map[string]interface{}{
		"nonInteractive": !interactiveConfig,
	}

	job, err := SendCommandAsync(
		"UI:Configuration", "Save '"+name.(string)+"'", command, "/config/"+mode,
	)
	if err != nil {
		return err
	}

	_, err = GetJobReply(job)

	return err
}

// DeleteConfig deletes the configuration.
func DeleteConfig(name string) error {
	command := map[string]interface{}{
		"name": name,
	}

	job, err := SendCommandAsync(
		"UI:Configuration", "Delete '"+name+"'", command, "/config/delete",
	)
	if err != nil {
		return err
	}

	_, err = GetJobReply(job)

	return err
}
