package loadbalancer

import ("Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
		"net/url"
)


type LoadBalancer interface {
	GetNextValidPeer() *backend.Backend
	AddBackend(backend *backend.Backend) error
	SetBackendStatus(uri *url.URL, alive bool)
	RemoveBackend(uri *url.URL) bool
}