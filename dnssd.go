/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netserver
	cmdCh chan *command
}

var ds *dnssd

func getDnssd() *dnssd {
	if ds == nil {
		ns, err := makeNetserver(nil)
		if err != nil {
			panic("Could not start netserver")
		}
		ns.startReceiving()
		cmdCh := make(chan *command, 32)
		ds = &dnssd{ns, cmdCh}
		go ds.processing()
	}
	return ds
}

func (ds *dnssd) processing() {
	var cs []*command
	rrc := &rrcache{} // Remote entries, lookup only
	rrl := &rrcache{} // Local entries, repond and lookup.

	for {
		select {
		case cmd := <-ds.cmdCh:
			if cmd.rr != nil {
				rrl.add(cmd.rr)
				resp := new(dns.Msg)
				resp.MsgHdr.Response = true
				resp.Answer = []dns.RR{cmd.rr}
				resp.Extra = []dns.RR{}
				go ds.ns.sendUnsolicitedMessage(resp)
			} else {
				if !rrc.matchQuestion(cmd) {
					// TODO: Don't resend queries!
					err := ds.ns.sendQuery(cmd.q)
					if err != nil {
						cmd.errc(err)
					} else {
						cs = append(cs, cmd)
					}
				}
			}
		case msg := <-ds.ns.msgCh:
			i := 0 // output index
			for _, cmd := range cs {
				if cmd.isValid() {
					rrc.matchAnswers(cmd, msg.Answer)
					rrc.matchAnswers(cmd, msg.Ns)
					rrc.matchAnswers(cmd, msg.Extra)
					if cmd.isValid() {
						cs[i] = cmd
						i++
					}
				}
			}
			cs = cs[:i]
		}
	}
}

// Shutdown server will close currently open connections & channel
func (ds *dnssd) shutdown() error {
	close(ds.cmdCh)
	return ds.ns.shutdown()
}
