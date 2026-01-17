package proxy

import (
	"lb/loadbalancer"
    "log"
    "net/http"
    "net/http/httputil"
)
type ProxyHandler struct{
	serverPool loadbalancer.LoadBalancer
}
func NewProxyHandler(sp loadbalancer.LoadBalancer) *ProxyHandler{
	return &ProxyHandler{serverPool: sp}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request){
	backend:=p.serverPool.GetNextValidPeer()
	if backend==nil{
		http.Error(w,"No available servers",http.StatusServiceUnavailable)
		return
	}
	backend.IncConns()
	defer backend.DecConns()

	proxy:=httputil.NewSingleHostReverseProxy(backend.URL)
	proxy.ErrorHandler=func(w http.ResponseWriter,req *http.Request,err error){
		log.Printf("Backend %s failed: %v",backend.URL,err)
        backend.SetAlive(false)
        http.Error(w, "Backend error",http.StatusBadGateway)
	}
	proxy.ServeHTTP(w,req)
	log.Printf("Req sent to %s",backend.URL)
}
