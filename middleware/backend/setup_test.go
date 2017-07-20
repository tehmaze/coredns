// +build etcd

package etcd

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/coredns/coredns/middleware/backend/msg"
	"github.com/coredns/coredns/middleware/pkg/dnsrecorder"
	"github.com/coredns/coredns/middleware/pkg/singleflight"
	"github.com/coredns/coredns/middleware/pkg/tls"
	"github.com/coredns/coredns/middleware/proxy"
	"github.com/coredns/coredns/middleware/test"

	etcdc "github.com/coreos/etcd/client"
	"github.com/mholt/caddy"
	"golang.org/x/net/context"
)

func init() {
	ctxt, _ = context.WithTimeout(context.Background(), etcdTimeout)
}

func newEtcdMiddleware() *Backend {
	ctxt, _ = context.WithTimeout(context.Background(), etcdTimeout)

	endpoints := []string{"http://localhost:2379"}
	tlsc, _ := tls.NewTLSConfigFromArgs()
	client, _ := newEtcdClient(endpoints, tlsc)

	etcd := &EtcdV2{
		Proxy:      proxy.NewLookup([]string{"8.8.8.8:53"}),
		PathPrefix: "skydns",
		Ctx:        context.Background(),
		Inflight:   &singleflight.Group{},
		Client:     client,
	}
	return &Backend{
		Zones:          []string{"skydns.test.", "skydns_extra.test.", "in-addr.arpa."},
		ServiceName:    "etcd",
		ServiceBackend: etcd,
	}
}

func set(t *testing.T, b *Backend, k string, ttl time.Duration, m *msg.Service) {
	d, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := msg.PathWithWildcard(k, b.ServiceBackend.(*EtcdV2).PathPrefix)
	b.ServiceBackend.(*EtcdV2).Client.Set(ctxt, path, string(d), &etcdc.SetOptions{TTL: ttl})
}

func delete(t *testing.T, b *Backend, k string) {
	path, _ := msg.PathWithWildcard(k, b.ServiceBackend.(*EtcdV2).PathPrefix)
	b.ServiceBackend.(*EtcdV2).Client.Delete(ctxt, path, &etcdc.DeleteOptions{Recursive: false})
}

func TestLookup(t *testing.T) {
	etc := newEtcdMiddleware()
	for _, serv := range services {
		set(t, etc, serv.Key, 0, serv)
		defer delete(t, etc, serv.Key)
	}

	for _, tc := range dnsTestCases {
		m := tc.Msg()

		rec := dnsrecorder.New(&test.ResponseWriter{})
		etc.ServeDNS(ctxt, rec, m)

		resp := rec.Msg
		sort.Sort(test.RRSet(resp.Answer))
		sort.Sort(test.RRSet(resp.Ns))
		sort.Sort(test.RRSet(resp.Extra))

		if !test.Header(t, tc, resp) {
			t.Logf("%v\n", resp)
			continue
		}
		if !test.Section(t, tc, test.Answer, resp.Answer) {
			t.Logf("%v\n", resp)
		}
		if !test.Section(t, tc, test.Ns, resp.Ns) {
			t.Logf("%v\n", resp)
		}
		if !test.Section(t, tc, test.Extra, resp.Extra) {
			t.Logf("%v\n", resp)
		}
	}
}

func TestSetupEtcd(t *testing.T) {
	tests := []struct {
		input              string
		shouldErr          bool
		expectedPath       string
		expectedEndpoint   string
		expectedErrContent string // substring from the expected error. Empty for positive cases.
	}{
		// positive
		{
			`etcd`, false, "skydns", "http://localhost:2379", "",
		},
		{
			`etcd skydns.local {
	endpoint localhost:300
}
`, false, "skydns", "localhost:300", "",
		},
		// negative
		{
			`etcd {
	endpoints localhost:300
}
`, true, "", "", "unknown property 'endpoints'",
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		etcd, _ /*stubzones*/, err := etcdParse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found %s for input %s", i, err, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
				continue
			}

			if !strings.Contains(err.Error(), test.expectedErrContent) {
				t.Errorf("Test %d: Expected error to contain: %v, found error: %v, input: %s", i, test.expectedErrContent, err, test.input)
				continue
			}
		}

		if !test.shouldErr && etcd.ServiceBackend.(*EtcdV2).PathPrefix != test.expectedPath {
			t.Errorf("EtcdV2 not correctly set for input %s. Expected: %s, actual: %s", test.input, test.expectedPath, etcd.ServiceBackend.(*EtcdV2).PathPrefix)
		}
		if !test.shouldErr && etcd.ServiceBackend.(*EtcdV2).endpoints[0] != test.expectedEndpoint { // only checks the first
			t.Errorf("EtcdV2 not correctly set for input %s. Expected: '%s', actual: '%s'", test.input, test.expectedEndpoint, etcd.ServiceBackend.(*EtcdV2).endpoints[0])
		}
	}
}

var ctxt context.Context
