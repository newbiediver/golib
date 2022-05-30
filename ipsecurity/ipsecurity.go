package ipsecurity

import (
	"github.com/jpillora/ipfilter"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
)

type IPSecurity struct {
	BlockByDefault bool
	TrustedProxies []string
	AllowIPs       []string
	BlockIPs       []string
	AllowCountries []string
	BlockCountries []string
}

type yamlEntry map[string]interface{}

var (
	currentHandler *IPSecurity
)

func NewIPSecurity() *IPSecurity {
	r := new(IPSecurity)
	currentHandler = r
	return r
}

func Handler() *IPSecurity {
	return currentHandler
}

type policyYaml struct {
	BasicInfo yamlEntry `yaml:"api"`
	AllowInfo yamlEntry `yaml:"allow"`
	BlockInfo yamlEntry `yaml:"block"`
}

func (ip *IPSecurity) LoadPolicyFile(policyYamlPath string) error {
	py := new(policyYaml)
	by, err := ioutil.ReadFile(policyYamlPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(by, py)
	if err != nil {
		return err
	}

	ip.applyYaml(py)

	return nil
}

func (ip *IPSecurity) applyYaml(ya *policyYaml) {
	if blockDefault := ya.BasicInfo["blockDefault"]; blockDefault != nil {
		ip.BlockByDefault = blockDefault.(bool)
	}

	if proxies := ya.BasicInfo["trusted"]; proxies != nil {
		ip.TrustedProxies = make([]string, len(proxies.([]interface{})))
		for i, cidr := range proxies.([]interface{}) {
			ip.TrustedProxies[i] = cidr.(string)
		}
	}

	if allowIPs := ya.AllowInfo["ips"]; allowIPs != nil {
		ip.AllowIPs = make([]string, len(allowIPs.([]interface{})))
		for i, ipString := range allowIPs.([]interface{}) {
			ip.AllowIPs[i] = ipString.(string)
		}
	}

	if allowCountries := ya.AllowInfo["countries"]; allowCountries != nil {
		ip.AllowCountries = make([]string, len(allowCountries.([]interface{})))
		for i, country := range allowCountries.([]interface{}) {
			ip.AllowCountries[i] = country.(string)
		}
	}

	if blockIPs := ya.BlockInfo["ips"]; blockIPs != nil {
		ip.BlockIPs = make([]string, len(blockIPs.([]interface{})))
		for i, ipString := range blockIPs.([]interface{}) {
			ip.BlockIPs[i] = ipString.(string)
		}
	}

	if blockCountries := ya.BlockInfo["countries"]; blockCountries != nil {
		ip.BlockCountries = make([]string, len(blockCountries.([]interface{})))
		for i, country := range blockCountries.([]interface{}) {
			ip.BlockCountries[i] = country.(string)
		}
	}
}

func (ip *IPSecurity) SetupIPSecurityPolicy(handler http.Handler) http.Handler {
	filter := ipfilter.New(ipfilter.Options{
		BlockByDefault:   ip.BlockByDefault,
		AllowedCountries: ip.AllowCountries,
		AllowedIPs:       ip.AllowIPs,
		BlockedIPs:       ip.BlockIPs,
		BlockedCountries: ip.BlockCountries,
		TrustProxy:       true,
		Logger:           log.Default(),
	})

	return filter.Wrap(handler)
}
