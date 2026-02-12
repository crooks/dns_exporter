package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"codeberg.org/miekg/dns"
	"github.com/Masterminds/log-go"
	"github.com/crooks/dns_exporter/config"
	loglevel "github.com/crooks/log-go-level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cfg   *config.Config
	flags *config.Flags
	prom  *prometheusMetrics
)

// LookupResult is the struct returned by each DNS lookup operation
type LookupResult struct {
	Success         bool
	NumAnswersA     int
	NumAnswersCNAME int
	Rtt             time.Duration // rtt = Round-trip time
}

// dnsALookup performs a lookup for a given domain at a list of Nameservers
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

	for _, answer := range r.Answer {
		_, rrA := answer.(*dns.A)
		if rrA {
			result.Success = true
			result.NumAnswersA++
		}
		_, rrCNAME := answer.(*dns.CNAME)
		if rrCNAME {
			result.NumAnswersCNAME++
		}
	}
	return
}

// iterDomains iterates through a configured list of Domains to lookup.  For each Domain iteration,
// it iterates through a configured series of Nameservers at which the query should be directed.
func iterDomains() {
	for dom, params := range cfg.Resolve {
		for _, ns := range params.Nameservers {
			result, err := dnsALookup(ns, dom)
			if err != nil {
				log.Warnf("Lookup Error for %s: %v", dom, err)
				prom.resolverResponse.WithLabelValues(ns).Set(0)
				continue
			}
			log.Debugf("Nameserver %s responded with RTT: %fs", ns, result.Rtt.Seconds())
			prom.resolverResponse.WithLabelValues(ns).Set(1)
			prom.resolverRTT.WithLabelValues(ns).Set(result.Rtt.Seconds())
			// Record a success here for the nameserver.  No error so it has responded.
			if result.Success {
				log.Debugf("Lookup of %s at %s successful", dom, ns)
				prom.lookupSuccess.WithLabelValues(ns, dom).Set(1)
			} else {
				log.Infof("Lookup of %s returned no answers", dom)
				prom.lookupSuccess.WithLabelValues(ns, dom).Set(0)
			}
			prom.lookupNumAnswers.WithLabelValues(ns, dom, "A").Set(float64(result.NumAnswersA))
			prom.lookupNumAnswers.WithLabelValues(ns, dom, "CNAME").Set(float64(result.NumAnswersCNAME))
		} // End of Nameservers loop
	} // End of Domains loop
}

// parseLoop performs an endless loop of queries at 60 second intervals.
func parseLoop() {
	log.Infof("Beginning iteration over %d domain lookups", len(cfg.Resolve))
	for {
		iterDomains()
		time.Sleep(60 * time.Second)
	}
}

// initServer triggers the DNS parsing process and starts the HTTP Server that will listen for
// incoming scrape requests.
func initServer() {
	go parseLoop()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`<html>
		<head><title>DNS Exporter</title></head>
		<body>
		<h1>DNS Exporter</h1>
		<p><a href='/metrics'>Metrics</a></p>
		</body>
		</html>`))
		if err != nil {
			log.Warnf("Error on returning homepage: %s", err)
		}
	})
	exporter := fmt.Sprintf("%s:%d", cfg.Exporter.Address, cfg.Exporter.Port)
	log.Infof("Starting HTTP server on: %s", exporter)
	err := http.ListenAndServe(exporter, nil)
	if err != nil {
		log.Fatalf("HTTP listener failed: %v", err)
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
	prom = initMetrics()
	initServer()
}
