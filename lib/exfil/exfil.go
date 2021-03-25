package exfil

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("exfil")

// Exfil implements a coredns plugin that is the receiver for
// dns exfiltrated data
type Exfil struct {
	Next      plugin.Handler
	suffix    string
	directory string
	receiver  *receiver
}

func (e Exfil) Name() string {
	return "exfil"
}

func replyRRs(qtype uint16, name string) []dns.RR {
	rrs := map[string]dns.RR{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if !ip.IsGlobalUnicast() {
				continue
			}
			if ip.To4() != nil && qtype == dns.TypeA {
				rrs[ip.String()] = &dns.A{
					Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
					A:   ip,
				}
			} else if qtype == dns.TypeAAAA {
				rrs[ip.String()] = &dns.AAAA{
					Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
					AAAA: ip,
				}
			}
		}
	}
	r := make([]dns.RR, 0, len(rrs)+1)
	for _, e := range rrs {
		r = append(r, e)
	}
	return r
}

func reply(w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	ns := new(dns.NS)
	ns.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 60}
	ns.Ns = state.QName()

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = append(replyRRs(state.QType(), state.QName()), ns)

	w.WriteMsg(m)

	return 0, nil
}

// ServeDNS takes the requests and extracts the payload
//
// Todo:
//  - indicate success or failure to the client
func (e Exfil) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	// log.Infof("QUERY: %s %s", dns.TypeToString[state.QType()], state.Name())

	switch state.QType() {
	default:
		return reply(w, r)
	case dns.TypeA, dns.TypeAAAA, dns.TypeNS:
		break
	}

	payload, ok := e.payload(state.Name())
	if !ok {
		return reply(w, r)
	}

	if payload == "" {
		return reply(w, r)
	}

	decoded, err := Decode(payload)
	if err == ERR_PAYLOAD_INCOMPLETE {
		return reply(w, r)
	}

	if err != nil {
		log.Warning("Invalid payload: ", err)
		return reply(w, r)
	}

	if decoded.IsHeader() {
		if err := e.receiver.AddFile(decoded.Id(), decoded.Header().Size(), decoded.Header().Name()); err != nil {
			log.Warning("AddFile failed: ", err)
		}
	} else {
		if err := e.receiver.AddData(decoded.Id(), decoded.Content().Offset(), decoded.Content().Data()); err != nil {
			log.Warning("AddData failed: ", err)
		}
	}

	return reply(w, r)
}

func (e Exfil) payload(s string) (string, bool) {
	if len(s) < len(e.suffix) {
		return "", false
	}
	if s == e.suffix {
		return "", true
	}
	if strings.HasSuffix(s, e.suffix) {
		return strings.TrimRight(s[0:len(s)-len(e.suffix)], "."), true
	}
	return "", false
}

func init() { plugin.Register("exfil", setup) }

func setup(c *caddy.Controller) error {
	e := Exfil{}

	for c.Next() {
		// syntax: exfil <domain suffix> <directory>
		if !c.NextArg() {
			return c.ArgErr()
		}
		e.suffix = c.Val()
		if !strings.HasSuffix(e.suffix, ".") {
			e.suffix += "."
		}
		if !c.NextArg() {
			return c.ArgErr()
		}
		e.directory = c.Val()
		e.receiver = NewReceiver(e.directory)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		e.Next = next
		return e
	})

	return nil
}
