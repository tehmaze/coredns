package geo

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/rainycape/geoip"
)

// Geo IP DNS plugin
type Geo struct {
	// Next plugin in the stack
	Next plugin.Handler

	// Config for our plugin
	Config *Config
}

// ServeDNS serves the DNS plugin response.
func (geo Geo) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	if state.QClass() != dns.ClassINET {
		return plugin.NextOrFailure(geo.Name(), geo.Next, ctx, w, r)
	}

	name := state.Name()

	// See if the name requested matches any of the domains we serve
	domain, ok := geo.LookupDomain(name)
	if !ok {
		// Not handling this domain; maybe the next backend is...
		return geo.Next.ServeDNS(ctx, w, r)
	}

	// See if the name is a direct hit for our static records
	rrs, ok := domain.Records[name]
	if !ok || len(rrs) == 0 {
		// No records, is there a service with this name?
		if service, ok := domain.Services[name]; ok {
			// There is a service, let's see to what records it resolves
			name = geo.Expand(service, w)
			if rrs, ok = domain.Records[name]; !ok {
				return 0, nil
			}
		} else {
			// Nothing to see here
			return 0, nil
		}
	}

	// Build response
	m := new(dns.Msg)
	m.SetReply(r)

	var (
		qtype = state.QType()
		any   = qtype == dns.TypeANY
	)
	for _, rr := range rrs {
		header := rr.Header()
		if any || header.Rrtype == qtype {
			header.Name = state.Name()
			m.Answer = append(m.Answer, rr)
		}
	}

	// Resolve authority section
	//
	// XXX(maze): this needs more love, and I'm most likely doing it wrong
	if name != state.Name() && (qtype == dns.TypeNS || qtype == dns.TypeSOA) {
		for _, rr := range domain.Records[domain.Domain] {
			if rr.Header().Rrtype == dns.TypeSOA {
				m.Ns = append(m.Ns, rr)
			}
		}
	}

	// Finalize answer
	state.SizeAndDo(m)
	w.WriteMsg(m)
	return 0, nil
}

// Expand a name using the IP information supplied by the dns.ResponseWriter
func (geo Geo) Expand(name string, w dns.ResponseWriter) string {
	var (
		record *geoip.Record
		host   string
		ip     net.IP
		now    = time.Now().UTC()
		err    error
	)

	if host, _, err = net.SplitHostPort(w.RemoteAddr().String()); err == nil {
		host = "77.81.229.156"
		if ip = net.ParseIP(host); ip != nil {
			if record, err = geo.Config.GeoIP.LookupIP(ip); err != nil {
				log.Printf("geo: error looking up %s: %v\n", ip, err)
			}
		}
	}
	if record != nil && record.Country.Code != "" {
		name = strings.Replace(name, "%co", strings.ToLower(record.Country.Code), -1)
	} else {
		name = strings.Replace(name, "%co", "unknown", -1)
	}
	if record != nil && record.Continent.Code != "" {
		name = strings.Replace(name, "%cn", strings.ToLower(record.Continent.Code), -1)
	} else {
		name = strings.Replace(name, "%cn", "unknown", -1)
	}
	if ip != nil && ip.To16() != nil {
		name = strings.Replace(name, "%af", "v6", -1)
	} else {
		name = strings.Replace(name, "%af", "v4", -1)
	}
	if strings.Contains(name, "%hh") {
		name = strings.Replace(name, "%hh", fmt.Sprintf("%02d", now.Hour()), -1)
	}
	if strings.Contains(name, "%yy") {
		name = strings.Replace(name, "%yy", fmt.Sprintf("%02d", now.Year()), -1)
	}
	if strings.Contains(name, "%dd") {
		name = strings.Replace(name, "%dd", fmt.Sprintf("%02d", now.YearDay()), -1)
	}
	if strings.Contains(name, "%wds") {
		name = strings.Replace(name, "%wds", weekDays[now.Weekday()], -1)
	}
	if strings.Contains(name, "%wd") {
		name = strings.Replace(name, "%wd", fmt.Sprintf("%02d", now.Weekday()), -1)
	}
	if strings.Contains(name, "%mos") {
		name = strings.Replace(name, "%mos", months[now.Month()-1], -1)
	}
	if strings.Contains(name, "%mo") {
		name = strings.Replace(name, "%mo", fmt.Sprintf("%02d", now.Month()), -1)
	}
	if strings.Contains(name, "%ip") {
		if ip != nil {
			name = strings.Replace(name, "%ip", ip.String(), -1)
		} else {
			name = strings.Replace(name, "%ip", "unknown", -1)
		}
	}
	return strings.Replace(name, "%%", "%", -1)
}

// LookupDomain looks up a domain by name or a domain that contains this name.
func (geo Geo) LookupDomain(name string) (*Domain, bool) {
	if !dns.IsFqdn(name) {
		name = dns.Fqdn(name)
	}
	var domains Domains
	for _, domain := range geo.Config.Domains {
		if domain.Domain == name || dns.IsSubDomain(domain.Domain, name) {
			domains = append(domains, domain)
		}
	}
	if len(domains) > 0 {
		// We're returning the most specific (or shortest) match
		sort.Sort(domains)
		return domains[0], true
	}
	return nil, false
}

// Name implements the Namer interface
func (Geo) Name() string {
	return "geo"
}

var (
	weekDays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	months   = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
)
