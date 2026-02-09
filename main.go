package main

import (
	"context"
	"strings"
	"time"

	"codeberg.org/miekg/dns"
	"github.com/Masterminds/log-go"
	"github.com/crooks/dns_exporter/config"
	loglevel "github.com/crooks/log-go-level"
)

var (
	cfg   *config.Config
	flags *config.Flags
)

type LookupResult struct {
	Success bool
	Rtt     time.Duration // rtt = Round-trip time
}

func dnsALookup(nameServer, hostName string) (result LookupResult, err error) {
	m := dns.NewMsg(hostName, dns.TypeA)
	m.ID = dns.ID()
	m.RecursionDesired = true

	c := new(dns.Client)
	r, rtt, err := c.Exchange(context.TODO(), m, "udp", nameServer)
	if err != nil {
		return
	}
	result.Rtt = rtt

	//if t, ok := r.Answer[0].(*dns.A); ok {
	if len(r.Answer) == 0 {
		return
	}
	result.Success = true
	return
}

func addDNSPort(hostname string) string {
	colonTest := strings.Split(hostname, ":")
	if len(colonTest) == 1 {
		return hostname + ":53"
	}
	return hostname
}

func iterDomains() {
	for dom, params := range cfg.Resolve {
		ns := addDNSPort(params.Nameserver)
		result, err := dnsALookup(ns, dom)
		if err != nil {
			log.Warnf("Lookup Error for %s: %v", dom, err)
			// Record a failure
			continue
		}
		log.Debugf("Nameserver %s responded with RTT: %fs", ns, result.Rtt.Seconds())
		// Record a success here for the nameserver.  No error so it has responded.
		if result.Success {
			log.Debugf("Lookup of %s at %s successful", dom, ns)
		} else {
			log.Infof("Lookup of %s returned no answers", dom)
		}
	}
}

func parseLoop() {
	log.Infof("Beginning iteration over %d domain lookups", len(cfg.Resolve))
	for {
		iterDomains()
		time.Sleep(60 * time.Second)
	}
}

func main() {
	var err error
	flags = config.ParseFlags()
	cfg, err = config.ParseConfig(flags.Config)
	if err != nil {
		log.Fatalf("Unable to parse config file: %v", err)
	}
	loglev, err := loglevel.ParseLevel(cfg.Logging.LevelStr)
	log.Current = log.StdLogger{Level: loglev}
	parseLoop()
}
