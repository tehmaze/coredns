package geo

import (
	"io/ioutil"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/mholt/caddy"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	caddy.RegisterPlugin("geo", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	config, err := geoParse(c)
	if err != nil {
		return plugin.Error("geo", err)
	}
	if err = config.Parse(); err != nil {
		return plugin.Error("geo", err)
	}
	if err = config.OpenDatabase(); err != nil {
		return plugin.Error("geo", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Geo{Next: next, Config: config}
	})

	return nil
}

func geoParse(c *caddy.Controller) (*Config, error) {
	geoConfig := new(Config)

	for c.Next() {
		if !c.NextArg() {
			return nil, c.ArgErr()
		}
		data, err := ioutil.ReadFile(c.Val())
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(data, geoConfig); err != nil {
			return nil, err
		}
		return geoConfig, nil
	}

	return nil, c.ArgErr()
}
