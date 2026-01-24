package proxy

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/loadbalancer"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)
type ProxyHandler struct{
	serverPool loadbalancer.LoadBalancer
	transport *http.Transport
}
func NewProxyHandler(sp loadbalancer.LoadBalancer) *ProxyHandler{
	transport := &http.Transport{
        DialContext:(&net.Dialer{
            Timeout:5*time.Second, //max time to establish connection
            KeepAlive:30*time.Second, //keep connections alive
        }).DialContext,
        MaxIdleConns:50, //reuse connections
        IdleConnTimeout:60*time.Second,
        ResponseHeaderTimeout:10*time.Second, //max wait for response headers
    }
	return &ProxyHandler{serverPool: sp,transport:transport,}
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
	proxy.Transport=p.transport	
	proxy.ErrorHandler=func(w http.ResponseWriter,req *http.Request,err error){
		log.Printf("Backend %s failed: %v",backend.URL,err)
        p.serverPool.SetBackendStatus(backend.URL, false)
        http.Error(w, "Backend error",http.StatusBadGateway)
	}
	
	proxy.ServeHTTP(w,req)
	log.Printf("Req sent to %s",backend.URL)
}
