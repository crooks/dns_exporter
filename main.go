package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

type rrAnswer struct {
	Name  string
	TTL   int
	Class string
	Type  string
	RData string
}

func answerFields(answer string) (rr rrAnswer, err error) {
	fields := strings.Fields(answer)
	if len(fields) != 5 {
		err = errors.New("unexpected field count in RR answer")
		return
	}
	rr.Name = fields[0]
	ttl, err := strconv.Atoi(fields[1])
	if err != nil {
		err = errors.New("TTL is not an integer")
		return
	}
	rr.TTL = ttl
	rr.Class = fields[2]
	rr.Type = fields[3]
	rr.RData = fields[4]
	return
}

// dnsALookup performs a lookup for a given domain at a list of Nameservers
func dnsALookup(nameServer, domain string) (err error) {
	m := dns.NewMsg(domain, dns.TypeA)
	m.ID = dns.ID()
	m.RecursionDesired = true

	c := new(dns.Client)
	r, rtt, err := c.Exchange(context.TODO(), m, "udp", nameServer)
	if err != nil {
		// If the nameserver doesn't respond, both the resolver and the lookup need to be marked as unsuccessful
		prom.resolverResponse.WithLabelValues(nameServer).Set(0)
		prom.lookupSuccess.WithLabelValues(nameServer, domain).Set(0)
		return
	}
	log.Debugf("Nameserver %s responded with RTT: %fs", nameServer, rtt.Seconds())
	prom.resolverResponse.WithLabelValues(nameServer).Set(1)
	prom.resolverRTT.WithLabelValues(nameServer).Set(rtt.Seconds())

	var numARecords int
	var numCNAMERecords int
	for _, answer := range r.Answer {
		/*
			rr, err := answerFields(answer.String())
			if err != nil {
				log.Warnf("Unable to decode RR answer: %v", err)
				continue
			}
		*/
		if _, rrA := answer.(*dns.A); rrA {
			// Only an A record is considered a success
			numARecords++
		}
		if _, rrCNAME := answer.(*dns.CNAME); rrCNAME {
			numCNAMERecords++
		}
	}
	prom.lookupNumAnswers.WithLabelValues(nameServer, domain, "A").Set(float64(numARecords))
	prom.lookupNumAnswers.WithLabelValues(nameServer, domain, "CNAME").Set(float64(numCNAMERecords))

	if numARecords > 0 {
		log.Debugf("Lookup of %s at %s successful", domain, nameServer)
		prom.lookupSuccess.WithLabelValues(nameServer, domain).Set(1)
	} else {
		log.Infof("Lookup of %s returned no A records", domain)
		prom.lookupSuccess.WithLabelValues(nameServer, domain).Set(0)
	}
	return
}

// iterDomains iterates through a configured list of Domains to lookup.  For each Domain iteration,
// it iterates through a configured series of Nameservers at which the query should be directed.
func iterDomains() {
	for dom, params := range cfg.Resolve {
		for _, ns := range params.Nameservers {
			err := dnsALookup(ns, dom)
			if err != nil {
				log.Warnf("Lookup Error for %s: %v", dom, err)
				continue
			}

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
