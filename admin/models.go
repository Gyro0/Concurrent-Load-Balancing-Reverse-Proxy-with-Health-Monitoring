package admin

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
)

type AddBackendRequest struct {
	URL string
}

type BackendInfo struct {
	URL          string
	Alive        bool
	CurrentConns int64
}

type StatusResponse struct {
	TotalBackends  int
	ActiveBackends int
	Backends       []BackendInfo
}

type AdminAPI struct {
	serverPool *servers.ServerPool
}