package derp

import (
	"encoding/json"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/go-multierror"
	"github.com/jsiebens/ionscale/internal/config"
	"os"
	"tailscale.com/tailcfg"
)

func LoadDERPSources(c *config.Config) (*tailcfg.DERPMap, error) {
	derpMap := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	var merr *multierror.Error
	for _, src := range c.DERP.Sources {
		dm, err := loadDERPSource(src)
		if err != nil {
			merr = multierror.Append(merr, err)
			continue
		}

		for id, r := range dm.Regions {
			derpMap.Regions[id] = r
		}
	}

	if !c.DERP.Server.Disabled {
		dm := c.DefaultDERPMap()
		for id, r := range dm.Regions {
			derpMap.Regions[id] = r
		}
	}

	return derpMap, merr.ErrorOrNil()
}

func loadDERPSource(src string) (*tailcfg.DERPMap, error) {
	temp, err := os.CreateTemp(os.TempDir(), "derp-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp.Name())

	if err := getter.Get(temp.Name(), src, getter.WithMode(getter.ClientModeFile)); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(temp.Name())
	if err != nil {
		return nil, err
	}

	var dm tailcfg.DERPMap

	if err := json.Unmarshal(content, &dm); err != nil {
		return nil, err
	}

	return &dm, nil
}
