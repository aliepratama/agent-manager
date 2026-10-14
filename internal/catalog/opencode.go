package catalog

import (
	"context"
	"net/http"
	"regexp"
	"sort"
)

var opencodeListening = regexp.MustCompile(`listening on (http://\S+)`)

// opencode's ACP server lists models only inside a session, which then
// stays in opencode's history, so the headless server is asked.
func readOpencode(ctx context.Context, command, dir string) (Catalog, error) {
	password := secret()
	proc, err := start(command, dir, true, "OPENCODE_SERVER_PASSWORD="+password)
	if err != nil {
		return Catalog{}, err
	}
	defer proc.stop()
	address, err := serverAddress(ctx, proc, opencodeListening)
	if err != nil {
		return Catalog{}, err
	}
	auth := func(request *http.Request) { request.SetBasicAuth("opencode", password) }
	var providers struct {
		Providers []struct {
			ID     string `json:"id"`
			Models map[string]struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"models"`
		} `json:"providers"`
	}
	if err := getJSON(ctx, address+"/config/providers", auth, &providers); err != nil {
		return Catalog{}, err
	}
	var config struct {
		Model string `json:"model"`
	}
	if err := getJSON(ctx, address+"/config", auth, &config); err != nil {
		return Catalog{}, err
	}
	var cat Catalog
	for _, provider := range providers.Providers {
		for _, model := range provider.Models {
			id := provider.ID + "/" + model.ID
			cat.Models = append(cat.Models, Model{ID: id, Label: model.Name, Default: id == config.Model})
		}
	}
	sort.Slice(cat.Models, func(i, j int) bool { return cat.Models[i].ID < cat.Models[j].ID })
	return cat, nil
}

func readOpencode2(ctx context.Context, command, dir string) (Catalog, error) {
	password := secret()
	proc, err := start(command, dir, true, "OPENCODE_SERVER_PASSWORD="+password)
	if err != nil {
		return Catalog{}, err
	}
	defer proc.stop()
	address, err := serverAddress(ctx, proc, opencodeListening)
	if err != nil {
		return Catalog{}, err
	}
	auth := func(request *http.Request) { request.SetBasicAuth("opencode", password) }
	var modelResp struct {
		Data []struct {
			ProviderID string `json:"providerID"`
			ModelID    string `json:"modelID"`
			Name       string `json:"name"`
		} `json:"data"`
	}
	if err := getJSON(ctx, address+"/api/model", auth, &modelResp); err != nil {
		return Catalog{}, err
	}
	var configList []struct {
		Info struct {
			Model struct {
				ProviderID string `json:"providerID"`
				Model      string `json:"model"`
			} `json:"model"`
		} `json:"info"`
	}
	defaultModel := ""
	if err := getJSON(ctx, address+"/api/config", auth, &configList); err == nil {
		for _, c := range configList {
			if c.Info.Model.ProviderID != "" && c.Info.Model.Model != "" {
				defaultModel = c.Info.Model.ProviderID + "/" + c.Info.Model.Model
				break
			}
		}
	}
	var cat Catalog
	for _, m := range modelResp.Data {
		id := m.ProviderID + "/" + m.ModelID
		cat.Models = append(cat.Models, Model{ID: id, Label: m.Name, Default: id == defaultModel})
	}
	sort.Slice(cat.Models, func(i, j int) bool { return cat.Models[i].ID < cat.Models[j].ID })
	return cat, nil
}

