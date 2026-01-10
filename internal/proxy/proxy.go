package proxy

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
)

var proxies []string

func Init() {
	list := os.Getenv("PROXY_LIST")
	if list == "" {
		return
	}

	for _, line := range strings.Split(list, ",") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Convert ip:port:user:pass to http://user:pass@ip:port
		parts := strings.Split(line, ":")
		if len(parts) == 4 {
			url := fmt.Sprintf("http://%s:%s@%s:%s", parts[2], parts[3], parts[0], parts[1])
			proxies = append(proxies, url)
		} else if len(parts) == 2 {
			// Simple ip:port format (no auth)
			url := fmt.Sprintf("http://%s:%s", parts[0], parts[1])
			proxies = append(proxies, url)
		}
	}
}

func GetRandom() string {
	if len(proxies) == 0 {
		return ""
	}
	return proxies[rand.Intn(len(proxies))]
}

func Count() int {
	return len(proxies)
}
