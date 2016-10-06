/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netserver
	cs    *questions
	cmdCh chan func()
	rrc   *answers
	rrl   *answers
}
