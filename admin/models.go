package admin

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
)

//AddBackendRequest representes the JSON body for adding or removing a backend
//its used in both POST and DELETE requests to /backends
type AddBackendRequest struct {
	URL string `json:"url"`
}

//BackendInfo contains information about a single backend server
//its used in the StatusResponse to provide details for all servers
type BackendInfo struct {
	URL          string `json:"url"`
	Alive        bool `json:"alive"`
	CurrentConns int64 `json:"current_connections"`
}

//StatusResponse is the complete response structure for GET requests
//it provides an overview of the entire server pool including total and active counts
type StatusResponse struct {
	TotalBackends  int `json:"total_backends"`
	ActiveBackends int `json:"active_backends"`
	Backends       []BackendInfo `json:"backends"`
}

//AdminAPI holds the admin API s dependencies and state
//it maintains a reference to the server pool to query and modify backend configurations
type AdminAPI struct {
	serverPool *servers.ServerPool
}