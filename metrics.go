package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prefix string = "dns"
)

type prometheusMetrics struct {
	resolverResponse *prometheus.GaugeVec
	resolverRTT      *prometheus.GaugeVec
	lookupSuccess    *prometheus.GaugeVec
	lookupNumAnswers *prometheus.GaugeVec
}

func addPrefix(s string) string {
	return fmt.Sprintf("%s_%s", prefix, s)
}

func initMetrics() *prometheusMetrics {
	defaultLabels := []string{"nameserver"}
	dns := new(prometheusMetrics)

	dns.lookupNumAnswers = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: addPrefix("lookup_num_answers"),
			Help: "The number of DNS Answers received for a given lookup.",
		},
		append(defaultLabels, "domain"),
	)
	prometheus.MustRegister(dns.lookupNumAnswers)

	dns.lookupSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: addPrefix("lookup_success"),
			Help: "Returns 1 (True) if the lookup returned at least 1 answer.",
		},
		append(defaultLabels, "domain"),
	)
	prometheus.MustRegister(dns.lookupSuccess)

	dns.resolverResponse = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: addPrefix("resolver_responded"),
			Help: "Returns 1 (True) if the DNS Resolver responded with an answer.",
		},
		defaultLabels,
	)
	prometheus.MustRegister(dns.resolverResponse)

	dns.resolverRTT = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: addPrefix("resolver_rtt"),
			Help: "The Round Trip Time in seconds for the DNS Resolver to respond.",
		},
		defaultLabels,
	)
	prometheus.MustRegister(dns.resolverRTT)

	return dns
}
