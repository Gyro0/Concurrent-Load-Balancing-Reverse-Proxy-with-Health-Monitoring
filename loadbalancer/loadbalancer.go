package loadbalancer

import ("lb/backend"
		"net/url"
)


type LoadBalancer interface {
	GetNextValidPeer() *backend.Backend
	AddBackend(backend *backend.Backend)
	SetBackendStatus(uri *url.URL, alive bool)
}