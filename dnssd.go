/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"fmt"
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
	rrc := &rrcache{}

	for {
		select {
		case cmd := <-c.cmdCh:
			//			fmt.Println("COMMAND: ", cmd)

			if !rrc.matchQuestion(cmd) {
				// TODO: Don't resend queries!
				fmt.Println("SEND-QUERY-COMMAND: ", cmd)
				err := c.sendQuery(cmd.q)
				if err != nil {
					respondWithError(cmd.r, err)
				} else {
					cs = append(cs, cmd)
				}
			}

		case msg := <-c.msgCh:
			sections := append(msg.Answer, msg.Ns...)
			sections = append(sections, msg.Extra...)
			for _, cmd := range cs {
				rrc.matchAnswers(cmd, sections)
			}
		}
	}
}
