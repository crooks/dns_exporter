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

type LookupResult struct {
	Success    bool
	NumAnswers int
	Rtt        time.Duration // rtt = Round-trip time
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
	result.NumAnswers = len(r.Answer)

	//if t, ok := r.Answer[0].(*dns.A); ok {
	if len(r.Answer) == 0 {
		return
	}
	result.Success = true
	return
}

func iterDomains() {
	for dom, params := range cfg.Resolve {
		ns := params.Nameserver
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
		}
		prom.lookupNumAnswers.WithLabelValues(ns, dom).Set(float64(result.NumAnswers))
	}
}

func parseLoop() {
	log.Infof("Beginning iteration over %d domain lookups", len(cfg.Resolve))
	for {
		iterDomains()
		time.Sleep(60 * time.Second)
	}
}

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
