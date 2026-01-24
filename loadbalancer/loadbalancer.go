package loadbalancer

import ("Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
		"net/url"
)

//LoadBalancer defines the interface that all load balancing strategies must implement
//this abstraction allows different algorithms (round-robin, least-connections) to be swapped easily
type LoadBalancer interface {
	//GetNextValidPeer selects and returns the next healthy backend server to handle a request
	GetNextValidPeer() *backend.Backend

	//AddBackend adds a new backend server to the load balancer pool
	//returns an error if the backend is a duplicate
	AddBackend(backend *backend.Backend) error

	//SetBackendStatus updates the health status of a backend
	SetBackendStatus(uri *url.URL, alive bool)

	// RemoveBackend removes a backend server from the pool
	RemoveBackend(uri *url.URL) bool
}