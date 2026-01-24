package proxy

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/loadbalancer"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

//ProxyHandler implements the http.Handler interface to forward requests to backend servers
//it uses the load balancer to select backends
//all incoming HTTP requests pass through this handler
type ProxyHandler struct{
	serverPool loadbalancer.LoadBalancer
	transport *http.Transport
}

//constructor that initializes a new proxy handler, sets up an optimized HTTP transport with
//connection pooling and timeouts, this settings improve performance by reusing connections
//instead of creating new ones
func NewProxyHandler(sp loadbalancer.LoadBalancer) *ProxyHandler{
	//configuring an http transport with optimized settings
	transport := &http.Transport{
        DialContext:(&net.Dialer{
            Timeout:5*time.Second, //max time to establish connection
            KeepAlive:30*time.Second, //keep connections alive
        }).DialContext,
        MaxIdleConns:50, //reuse connections
        IdleConnTimeout:60*time.Second,
        ResponseHeaderTimeout:10*time.Second, //max wait for response headers
    }
	//return the configured proxy handler
	return &ProxyHandler{serverPool: sp,transport:transport,}
}

//ServeHTTP handles incoming HTTP requests by forwarding them to a selected backend
//this method is called for every request that comes into the load balancer
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request){
	//selecting a healthy backend
	backend:=p.serverPool.GetNextValidPeer()
	if backend==nil{
		//no healthy backends available -> return 503 service unavailable
		http.Error(w,"No available servers",http.StatusServiceUnavailable)
		return
	}
	//increment the connection counter for this backend this is for least-connections algorithm
	backend.IncConns()
	//decrement after completing the request
	defer backend.DecConns()

	//create a reverse proxy specifically for this backend server
	proxy:=httputil.NewSingleHostReverseProxy(backend.URL)
	//use our custom transport
	proxy.Transport=p.transport
	//define a custom error handler for failures
	proxy.ErrorHandler=func(w http.ResponseWriter,req *http.Request,err error){
		log.Printf("Backend %s failed: %v",backend.URL,err)
		//mark the backend as down so it wont receive more requests
        p.serverPool.SetBackendStatus(backend.URL, false)
		//return 502 bad gateway to the client indicating backend failure
        http.Error(w, "Backend error",http.StatusBadGateway)
	}
	//forward the request to the selected backend and relay the response
	proxy.ServeHTTP(w,req)
	log.Printf("[PROXY] Req sent to %s",backend.URL)
}
