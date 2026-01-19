package main

import (
    "fmt"
    "Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/config"
    "Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/healthcheck"
    "Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/proxy"
    "Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
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
        log.Printf("Load Balancer listening on %s", address)
        log.Println("Ready to accept requests!")
        if err:=http.ListenAndServe(address, proxyHandler); err!=nil{
            log.Fatalf("Failed to start load balancer: %v",err)
        }
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