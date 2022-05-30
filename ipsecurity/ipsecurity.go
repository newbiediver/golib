package ipsecurity

/*
# ipfilter policy yaml 샘플

api:
  blockDefault: true	# 기본적으로 모든 ip 를 block할지 안할지 여부
  trusted:
  - 10.244.0.0/16		# 신뢰하는 프록시 서버의 대역 (예> docker network 대역. k8s container network 대역)
allow:					# blockDefault가 true 일 경우 참조됨
  ips:					# 화이트리스트 ip
  - 10.244.0.0/16		# 신뢰하는 프록시와 같은 내용을 쓸 경우는 같은 대역에서 접속을 허용해야핧 때..(예> k8s 클러스터 내의 pod 끼리 통신)
  - 59.9.184.33
  countries:			# 화이트리스트 국가
  - KR
  - US
block:					# blockDefault가 false 일 경우 참조됨
  ips:					# 블랙리스트 ip
  - 210.103.86.151
  - 162.53.177.0/24
  countries:			# 블랙리스트 국가
  - CN	# 짱깨
  - AF	# 아프간
  - KP	# 북한

*/

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
