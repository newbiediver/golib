package upnp

import (
	"errors"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"net"
	"net/url"
	"strings"
)

type InternetGatewayDevice interface {
	ExternalIP() (string, error)
	AddPortForward(internalPort, externalPort uint16, description, protocol string) error
	DeletePortForward(externalPort uint16, protocol string) error
	GetPortMappedList(index uint16) (uint16, error)
}

type upnpDevice struct {
	client interface {
		GetExternalIPAddress() (string, error)
		AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error
		DeletePortMapping(string, uint16, string) error
		GetServiceClient() *goupnp.ServiceClient
		GetGenericPortMappingEntry(uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error)
	}
}

func Discover() (InternetGatewayDevice, error) {
	pppClients, _, _ := internetgateway2.NewWANPPPConnection1Clients()
	if len(pppClients) > 0 {
		return &upnpDevice{pppClients[0]}, nil
	}

	ipClients, _, _ := internetgateway2.NewWANIPConnection1Clients()
	if len(ipClients) > 0 {
		return &upnpDevice{ipClients[0]}, nil
	}

	return nil, errors.New("no upnp-enabled gateway found")
}

func DiscoverByRouterURL(rawUrl string) (InternetGatewayDevice, error) {
	loc, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	pppClients, _ := internetgateway2.NewWANPPPConnection1ClientsByURL(loc)
	if len(pppClients) > 0 {
		return &upnpDevice{pppClients[0]}, nil
	}

	ipClients, _ := internetgateway2.NewWANIPConnection1ClientsByURL(loc)
	if len(ipClients) > 0 {
		return &upnpDevice{ipClients[0]}, nil
	}

	return nil, errors.New("no upnp-enabled gateway found at URL " + rawUrl)
}

func (u *upnpDevice) getInternalIP() (string, error) {
	host, _, _ := net.SplitHostPort(u.client.GetServiceClient().RootDevice.URLBase.Host)
	ip := net.ParseIP(host)
	if ip == nil {
		return "", errors.New("could not determine router's internal IP")
	}

	lanCards, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, lan := range lanCards {
		addrs, err := lan.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			switch x := addr.(type) {
			case *net.IPNet:
				if x.Contains(ip) {
					return x.IP.String(), nil
				}
			}
		}
	}

	return "", errors.New("could not determine internal IP")
}

func (u *upnpDevice) ExternalIP() (string, error) {
	return u.client.GetExternalIPAddress()
}

func (u *upnpDevice) GetPortMappedList(index uint16) (uint16, error) {
	_, existExternalPort, _, _, _, _, _, _, err := u.client.GetGenericPortMappingEntry(index)
	return existExternalPort, err
}

func (u *upnpDevice) AddPortForward(internalPort, externalPort uint16, description, protocol string) error {
	proto := strings.ToUpper(protocol)
	ip, err := u.getInternalIP()
	if err != nil {
		return err
	}

	return u.client.AddPortMapping("", externalPort, proto, internalPort, ip, true, description, 0)
}

func (u *upnpDevice) DeletePortForward(externalPort uint16, protocol string) error {
	proto := strings.ToUpper(protocol)
	return u.client.DeletePortMapping("", externalPort, proto)
}
