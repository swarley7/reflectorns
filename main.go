package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"
)

// var records = map[string]string{
// 	"test.service.": "192.168.0.2",
// }

var domain string
var port string
var host string
var domParts []string

func parseHostname(hostname string) map[string]string {
	out := make(map[string]string)
	out["req"] = hostname
	out["domain"] = domain
	reqParts := strings.Split(hostname, ".") // get all the parts of the dns request
	if len(reqParts) < len(domParts)+4 {
		out["ip"] = ""
		log.Println("Broken request for:", hostname, "Not enough parts to construct IP")
		return out
	}
	ipParts := reqParts[len(reqParts)-len(domParts)-4 : len(reqParts)-len(domParts)]
	wildcardTrash := reqParts[:len(reqParts)-len(domParts)-4]
	out["ip"] = strings.Join(ipParts, ".")
	out["wildcard"] = strings.Join(wildcardTrash, ".")
	log.Printf("Responded: [Request: \"%v\" IP: \"%s\"]", out["req"], out["ip"])
	return out
}

func parseQuery(m *dns.Msg) {
	var ip string
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)
			q.Name = strings.Trim(q.Name, ".")
			if q.Name == domain { // Query for root domain
				ip = host
			} else {
				parsed := parseHostname(q.Name)
				ip = parsed["ip"]
			}

			// ip := records[q.Name]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

func main() {
	// attach request handler func
	flag.StringVar(&host, "listen", "127.0.0.1", "Host to listen on")
	flag.StringVar(&port, "port", "5353", "Port to listen on")
	flag.StringVar(&domain, "domain", "example.com", "Domain to respond to")

	flag.Parse()
	domParts = strings.Split(domain, ".")
	dns.HandleFunc(domain, handleDnsRequest)

	// start server
	server := &dns.Server{Addr: host + ":" + port, Net: "udp"}
	log.Printf("Starting at %s\n", host+":"+port)
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
