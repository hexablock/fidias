package main

import (
	"fmt"
	"net"
	"strconv"
)

// given a advertise and bind address return the advertise addr or an error
func buildAdvertiseAddr(a, b string) (string, error) {
	var addr string
	if a != "" {
		addr = a
	} else {
		// Used bind if adv is not supplied
		addr = b
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	var ipaddr *net.IPAddr
	ipaddr, err = net.ResolveIPAddr("ip", host)
	if err == nil {
		ip := ipaddr.String()

		switch ip {
		case "", "0.0.0.0", "0:0:0:0:0:0:0:0", ":", "::":
			goto INVALID_ADDR
		}

		if port == "" {
			goto INVALID_ADDR
		}

		return ip + ":" + port, nil
	}

INVALID_ADDR:
	return "", fmt.Errorf("Invalid advertise address: %s", addr)
}

func parseAddr(host string) (string, int) {
	host, port, _ := net.SplitHostPort(host)
	i, _ := strconv.ParseInt(port, 10, 32)
	return host, int(i)
}
