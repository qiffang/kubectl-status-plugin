package clusterstatus

import (
	"bufio"
	"os"
	"strings"
)

func EnableBridgeNfCallIptables() bool{
	f, err := os.Open("/proc/sys/net/bridge/bridge-nf-call-iptables")
	if err != nil {
		return false
	}

	defer f.Close()
	scanner := bufio.NewScanner(f)

	enabled := false
	for (scanner.Scan()) {
		if strings.TrimSpace(scanner.Text()) == "1" {
			enabled = true
		}
	}

	return enabled
}