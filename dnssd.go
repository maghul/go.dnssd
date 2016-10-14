/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

var ns *netserver

func getNetserver() *netserver {
	if ns == nil {
		var err error
		ns, err = newNetserver(nil)
		if err != nil {
			panic("Could not start netserver")
		}
		ns.startReceiving()
		go ns.processing()
	}
	return ns
}

func (c *netserver) processing() {
	var cs []*command
	rrc := &rrcache{} // Remote entries, lookup only
	rrl := &rrcache{} // Local entries, repond and lookup.

	for {
		select {
		case cmd := <-c.cmdCh:
			fmt.Println("COMMAND: ", cmd)

			if cmd.rr != nil {
				fmt.Println("COMMAND:RR: ", cmd)
				rrl.add(cmd.rr)
				resp := new(dns.Msg)
				resp.MsgHdr.Response = true
				resp.Answer = []dns.RR{cmd.rr}
				resp.Extra = []dns.RR{}
				go c.sendUnsolicitedMessage(resp)
			} else {
				if !rrc.matchQuestion(cmd) {
					// TODO: Don't resend queries!
					fmt.Println("SEND-QUERY-COMMAND: ", cmd)
					err := c.sendQuery(cmd.q)
					if err != nil {
						cmd.errc(err)
					} else {
						cs = append(cs, cmd)
					}
				}
			}
		case msg := <-c.msgCh:
			i := 0 // output index
			for j, cmd := range cs {
				if cmd.isValid() {
					rrc.matchAnswers(cmd, msg.Answer)
					rrc.matchAnswers(cmd, msg.Ns)
					rrc.matchAnswers(cmd, msg.Extra)
					if cmd.isValid() {
						fmt.Println("len=", len(cs), ", i=", i, ", j=", j, " cmd=", cmd)
						cs[i] = cmd
						i++
					}
				}
			}
			fmt.Println("len1=", len(cs), "i=", i)
			cs = cs[:i]
			fmt.Println("len2=", len(cs))
		}
	}
}
