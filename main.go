package main

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/admin"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/config"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/healthcheck"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/proxy"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func main() {
    //load configuration
    cfg:=config.LoadConfig()
    log.Printf("Starting Load Balancer on port %d with strategy: %s",cfg.Port,cfg.Strategy)

    //create server pool and load backends
    serverPool:=servers.NewServerPool()
    serverPool.LoadBackends()

    //start backend servers in background
    go servers.RunServers(serverPool)
    log.Println("Backend servers starting...")

    //start health checker routine
    healthcheck,err:=healthcheck.NewHealthChecker(serverPool,cfg.HealthCheckFreq)
    if err!=nil{
        log.Fatalf("Failed to start health checker: %v",err)
    }    
    log.Println("Health Checker service starting...")

    healthcheck.Start()

    //create proxy handler
    proxyHandler:=proxy.NewProxyHandler(serverPool)

    //start load balancer
    address:=fmt.Sprintf(":%d",cfg.Port)
    go func() {
        log.Println("Starting LoadBalancer")
        err:=http.ListenAndServe(address, proxyHandler)
        if err!=nil{
            log.Fatalf("Failed to start load balancer: %v",err)
        }
        log.Printf("Load Balancer listening on %s", address)
        log.Println("Ready to accept requests!")
    }()
    
    //start admin server
    adminAPI:=admin.NewAdminAPI(serverPool)
    adminMux:=http.NewServeMux()
    adminMux.HandleFunc("/status",adminAPI.HandleGet)
    adminMux.HandleFunc("/backends",func(w http.ResponseWriter,req *http.Request){
        switch req.Method{
        case http.MethodPost:
            adminAPI.HandlePost(w,req)
        case http.MethodDelete:
            adminAPI.HandleDelete(w,req)
        default:
            http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
        }
    })
    adminServer:=&http.Server{
        Addr:fmt.Sprintf(":%d",cfg.AdminPort),
        Handler: adminMux,
        ReadTimeout: 7*time.Second,
        WriteTimeout: 7*time.Second,
    }

    go func(){
        log.Println("Starting Admin Server")
        err:=adminServer.ListenAndServe()
        if err!=nil{
            log.Fatalf("Failed to start admin server: %v", err)
        }
        log.Printf("Admin listening on %s", adminServer.Addr)
        log.Println("Ready to accept requests!")
    }()

    //test healthchecker working
    time.Sleep(5*time.Second)
    backendURL,_:=url.Parse("http://localhost:8084")
    serverPool.SetBackendStatus(backendURL,false)
    //health checker will set the backend status to true again(since it is up)
    time.Sleep(5*time.Second)
    backendURL2,_:=url.Parse("http://localhost:8083")
    serverPool.SetBackendStatus(backendURL2,false)
    select {}
}