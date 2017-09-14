package geo

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/rainycape/geoip"
)

// Config is our main plugin configuration structure.
type Config struct {
	// Domains is a slice of all configured domains.
	Domains []*Domain `yaml:"domains"`

	// Database is the file name of the Maxmind GeoIP2 database.
	Database string `yaml:"database"`

	// GeoIP instance, configured by Config.OpenDatabase()
	GeoIP *geoip.GeoIP `yaml:"-"`
}

// Domain configuration
type Domain struct {
	// Domain name.
	Domain string `yaml:"domain"`

	// TTL is the default time to live.
	TTL int `yaml:"ttl"`

	// RawRecords are the text records from the configuration, keyed by
	// domain name. The keys should be a subdomain of the domain.
	RawRecords map[string][]map[string]string `yaml:"records"`

	// Services are the service-to-records mapping.
	Services map[string]string `yaml:"services"`

	// Records are the parsed records, after Domain.Parse() has been called.
	Records map[string][]dns.RR `yaml:"-"`
}

// OpenDatabase opens the Maxmind GeoIP2 database.
func (c *Config) OpenDatabase() error {
	var err error
	if c.GeoIP, err = geoip.Open(c.Database); err != nil {
		return fmt.Errorf("error opening Maxmind Database: %v", err)
	}
	return nil
}

// Parse the configuration and all its sections.
func (c *Config) Parse() error {
	for _, domain := range c.Domains {
		if err := domain.Parse(); err != nil {
			return err
		}
	}
	return nil
}

// Parse the domain structure and all its records. This also takes care of
// rectifying zone information where applicable.
func (d *Domain) Parse() error {
	// Rectify domain name
	if !dns.IsFqdn(d.Domain) {
		d.Domain = dns.Fqdn(d.Domain)
	}

	// Parse and rectify domains and their rrs
	d.Records = make(map[string][]dns.RR)
	for name, rawRecords := range d.RawRecords {
		if !dns.IsFqdn(name) {
			name = dns.Fqdn(name)
		}
		if !dns.IsSubDomain(d.Domain, name) {
			return fmt.Errorf("record %q is not a subdomain of %q", name, d.Domain)
		}
		var records []dns.RR
		for _, rawRecord := range rawRecords {
			for rrType, content := range rawRecord {
				if strings.ToUpper(rrType) == "TXT" {
					content = fmt.Sprintf(`"%s"`, strings.Replace(content, `"`, `\"`, -1))
				}
				s := fmt.Sprintf("%s %d IN %s %s", d.Domain, d.TTL, strings.ToUpper(rrType), content)
				r, err := dns.NewRR(s)
				if err != nil {
					return err
				}
				records = append(records, r)
			}
		}
		d.Records[name] = records
	}

	// Rectify service names
	for name, service := range d.Services {
		if !dns.IsFqdn(name) {
			delete(d.Services, name)
			name = dns.Fqdn(name)
			d.Services[name] = service
		}
		if !dns.IsFqdn(service) {
			d.Services[name] = dns.Fqdn(service)
		}
		if !dns.IsSubDomain(d.Domain, name) {
			return fmt.Errorf("service %q is not a subdomain of %q", name, d.Domain)
		}
	}

	return nil
}

// Domains is a slice of Domain implementing the sort.Sorter interface.
type Domains []*Domain

// Len is the length of the slice.
func (ds Domains) Len() int {
	return len(ds)
}

// Less compares two slice values.
func (ds Domains) Less(i, j int) bool {
	return len(ds[i].Domain) < len(ds[j].Domain)
}

// Swap two slice values.
func (ds Domains) Swap(i, j int) {
	ds[i], ds[j] = ds[j], ds[i]
}
