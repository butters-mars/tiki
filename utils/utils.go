package utils

import (
	"net"

	"github.com/tiki/logging"
)

var logger = logging.Logger

// GetIP returns the first non-lo ip of this machine
func GetIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		logger.Error("fail to list ifaces", err)
		return ""
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		if i.Name == "lo" {
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
			return ip.String()
		}
	}

	return ""
}
