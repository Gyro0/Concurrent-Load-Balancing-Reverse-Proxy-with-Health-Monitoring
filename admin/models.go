package admin

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
)

type AddBackendRequest struct {
	URL string `json:"url"`
}

type BackendInfo struct {
	URL          string `json:"url"`
	Alive        bool `json:"alive"`
	CurrentConns int64 `json:"current_connections"`
}

type StatusResponse struct {
	TotalBackends  int `json:"total_backends"`
	ActiveBackends int `json:"active_backends"`
	Backends       []BackendInfo `json:"backends"`
}

type AdminAPI struct {
	serverPool *servers.ServerPool
}