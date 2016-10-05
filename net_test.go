package dnssd

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

func makeTestNetservers() (ns *netservers, err error) {
	msgCh := make(chan *incomingMsg, 32)

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	ns = &netservers{make(map[int]*netserver), msgCh}

	for _, iface := range ifaces {
		ns.addTestInterface(iface)
		if err != nil {
			return
		}
	}
	return
}

func (nss *netservers) addTestInterface(iface net.Interface) (err error) {
	ns, err := makeTestNetserver(iface, nss.msgCh)
	if err == nil {
		fmt.Println("addTestInterface: ifIndex=", iface.Index, ", ns=", ns)
		nss.servers[iface.Index] = ns
	} else {
		fmt.Println("ifIndex=", iface.Index, ", error=", err)
	}
	return
}

func makeTestNetserver(iface net.Interface, msgCh chan *incomingMsg) (*netserver, error) {

	ns := &netserver{iface: iface, msgCh: msgCh}
	ns.response = &dns.Msg{}
	ns.query = &dns.Msg{}
	return ns, nil
}
