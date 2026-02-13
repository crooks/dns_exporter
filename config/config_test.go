package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig(t *testing.T) {
	testCfg := `---
exporter:
  address: 127.0.0.1
  port: 9012
logging:
  level: trace
default_ns: 99.99.99.99
resolve:
  subdom.dom.test:
    nameservers:
	  - 10.11.12.13
  dom.foo:
    nameserver:
	  - 1.2.3.4
  nonameserver.com:
`
	var err error
	cfg := new(Config)
	if err = yaml.Unmarshal([]byte(testCfg), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Exporter.Address != "127.0.0.1" {
		t.Fatalf("Expected=127.0.0.1, Got=%s", cfg.Exporter.Address)
	}
	if cfg.Exporter.Port != 9012 {
		t.Fatalf("Expected=9012, Got=%d", cfg.Exporter.Port)
	}
	if cfg.Logging.Journal {
		t.Fatal("Expected Logging.Journal to be False")
	}
	if cfg.Logging.LevelStr != "trace" {
		t.Fatalf("Unexpected Logging.Level: Expected=trace, Got=%s", cfg.Logging.LevelStr)
	}
}

func TestFlags(t *testing.T) {
	f := ParseFlags()
	expectingConfig := "config.yml"
	if f.Config != expectingConfig {
		t.Fatalf("Unexpected config flag: Expected=%s, Got=%s", expectingConfig, f.Config)
	}
	if f.Debug {
		t.Fatal("Unexpected debug flag: Expected=false, Got=true")
	}
}
