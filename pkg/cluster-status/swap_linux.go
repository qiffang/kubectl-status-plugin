package clusterstatus

import (
	"bufio"
	"os"
	"strings"
)

func SwapDisabled() (bool, error) {
	f, err := os.Open("/proc/swaps")
	if err != nil {
		return false, err
	}

	defer f.Close()
	scanner := bufio.NewScanner(f)

	enabled := false
	for (scanner.Scan()) {
		if strings.Contains(strings.ToUpper(scanner.Text()), "Filename") {
			continue
		}

		enabled = true
	}

	return enabled, nil
}
