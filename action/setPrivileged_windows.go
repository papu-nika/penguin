package penguin

import ping "github.com/prometheus-community/pro-bing"

func SetPrivileged(p *ping.Pinger) {
	p.SetPrivileged(true)
}
