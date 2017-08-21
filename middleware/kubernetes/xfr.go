package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coredns/coredns/middleware/pkg/dnsutil"

	"github.com/miekg/dns"
	"k8s.io/client-go/1.5/pkg/api"
)

type Xfr struct {
	*Kubernetes
	sync.RWMutex
	epoch time.Time
}

// ServeDNS implements the middleware.Handler interface.
func (x *Xfr) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	/*
		state := request.Request{W: w, Req: r}
		if transfer.Allowed(state, x.TransferTo) {
			return dns.RcodeServerFailure, nil
		}
		if state.QType() != dns.TypeAXFR && state.QType() != dns.TypeIXFR {
			return 0, middleware.Error(x.Name(), fmt.Errorf("xfr called with non transfer type: %d", state.QType()))
		}

		records := x.All()
		if len(records) == 0 {
			return dns.RcodeServerFailure, nil
		}

		log.Printf("[INFO] Outgoing transfer of %d records of zone %s to %s started", len(records), x.origin, state.IP())
		// get soa record
		//	records = append(records, records[0]) // add closing SOA to the end

		//	transfer.Out(state, records)
	*/

	return dns.RcodeSuccess, nil
}

// Name implements the middleware.Hander interface.
func (x *Xfr) Name() string { return "xfr" }

func NewXfr(k *Kubernetes) *Xfr {
	return &Xfr{Kubernetes: k, epoch: time.Now().UTC()}
}

// All returns all kubernetes records with a SOA at the start.
func (x *Xfr) All(zone string) []dns.RR {

	// This is super expensive as we use dns.NewRR to create the RRs.

	res := []dns.RR{}

	serviceList := x.APIConn.ServiceList()
	for _, svc := range serviceList {

		name := dnsutil.Join([]string{svc.Name, svc.Namespace, zone})

		// Endpoint query or headless service
		if svc.Spec.ClusterIP == api.ClusterIPNone {

			endpointsList := x.APIConn.EndpointsList()
			for _, ep := range endpointsList.Items {
				if ep.ObjectMeta.Name != svc.Name || ep.ObjectMeta.Namespace != svc.Namespace {
					continue
				}
				for _, eps := range ep.Subsets {
					for _, addr := range eps.Addresses {
						for _, p := range eps.Ports {

							fmt.Printf("%T\n", addr.IP)
							fmt.Printf("%s IN A %s\n", name, addr.IP)
							fmt.Printf("_%s._%s.%s IN SRV %d %s.%s\n", p.Name, p.Protocol, name, p.Port, endpointHostname(addr), name)
							fmt.Printf("%s.%s IN A %s\n", endpointHostname(addr), name, addr.IP)
						}
					}
				}
			}
			continue
		}

		// External service
		if svc.Spec.ExternalName != "" {
			fmt.Printf("%s IN CNAME %s", name, svc.Spec.ExternalName)
			continue
		}

		// ClusterIP service
		fmt.Printf("%s IN A %s\n", name, svc.Spec.ClusterIP)
		for _, p := range svc.Spec.Ports {
			fmt.Printf("_%s._%s.%s IN SRV %s\n", p.Name, p.Protocol, name, name)
		}
	}
	return res
}

func (x *Xfr) serial() uint32 {
	x.RLock()
	defer x.RUnlock()
	return uint32(x.epoch.Unix())
}

// Give these to dnscontroller via the options, so these functions get exectuted and the SOA's serial gets updated.
func (x *Xfr) AddDeleteXfrHandler(a interface{}) {
	x.Lock()
	defer x.Unlock()
	x.epoch = time.Now().UTC()
}

func (x *Xfr) UpdateXfrHandler(a, b interface{}) {
	x.Lock()
	defer x.Unlock()
	x.epoch = time.Now().UTC()
}

/*
cache.ResourceEventHandlerFuncs{
    AddFunc: x.AddDeleteXfrHandler,
    DeleteFunc: x.AddDeleteXfrHandler,
    UpdateFunc: x.UpdateXfrHandler,
}

// set to nil? Or noop functions as defaults?
*/
