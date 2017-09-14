package geo

import (
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	config := new(Config)
	if err := yaml.Unmarshal([]byte(testConfig), config); err != nil {
		t.Fatal(err)
	}
	if err := config.Parse(); err != nil {
		t.Fatal(err)
	}

	if l := len(config.Domains); l != 1 {
		t.Fatalf("expected 1 domain, got %d", l)
	}

	domain := config.Domains[0]
	if domain.Domain != "hose.brandmeister.network." {
		t.Fatalf("unexpected domain %q", domain.Domain)
	}
	if domain.TTL != 60 {
		t.Fatalf("unexpected TTL %d", domain.TTL)
	}

	if l := len(domain.Records); l != 13 {
		t.Fatalf("expected 13 records, got %d", l)
	}

	if l := len(domain.Services); l != 1 {
		t.Fatalf("expected 1 service, got %d", l)
	}
}

const testConfig = `
domains:
- domain: hose.brandmeister.network
  ttl: 60
  records:
    hose.brandmeister.network:
      - soa: dns1.maze.io. systems-dns.maze.io. 666 7200 3600 1209600 3600
      - ns: dns1.maze.io.
      - ns: dns2.maze.io.
      - ns: dns3.maze.io.
      - a: 46.105.144.164
    it.hose.brandmeister.network:
      - a: 77.81.229.156
    ru.hose.brandmeister.network:
      - a: 44.188.129.251
    us.hose.brandmeister.network:
      - a: 74.91.118.233
      - a: 63.251.20.56
    nl.hose.brandmeister.network:
      - a: 46.105.144.164
    unknown.hose.brandmeister.network:
      - a: 44.188.129.251
      - a: 77.81.229.156
      - a: 46.105.144.164
      - txt: "Hose line"
    af.hose.brandmeister.network:
      - a: 77.81.229.156
      - a: 46.105.144.164
      - txt: "Hose line (Africa via Europe)"
    an.hose.brandmeister.network:
      - a: 74.91.118.233
      - a: 63.251.20.56
      - a: 162.248.92.62
      - txt: "Hose line (Antartica via North America)"
    as.hose.brandmeister.network:
      - a: 44.188.129.251
      - a: 77.81.229.156
      - txt: "Hose line (Asia via Europe)"
    eu.hose.brandmeister.network:
      - a: 46.105.144.164
      - a: 77.81.229.156
      - txt: "Hose line (Europe)"
    na.hose.brandmeister.network:
      - a: 74.91.118.233
      - a: 63.251.20.56
      - a: 162.248.92.62
      - txt: "Hose line (North America)"
    oc.hose.brandmeister.network:
      - a: 74.91.118.233
      - a: 63.251.20.56
      - a: 162.248.92.62
      - txt: "Hose line (Oceania via North America)"
    sa.hose.brandmeister.network:
      - a: 74.91.118.233
      - a: 63.251.20.56
      - a: 162.248.92.62
      - txt: "Hose line (South America via North America)"
  services:
    geo.hose.brandmeister.network: '%cn.hose.brandmeister.network'
`
