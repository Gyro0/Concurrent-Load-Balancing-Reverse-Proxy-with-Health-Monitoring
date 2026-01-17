package main

import (
    "fmt"
    "lb/config"
    "lb/proxy"
    "lb/servers"
    "log"
    "net/http"
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

    //create proxy handler
    proxyHandler:=proxy.NewProxyHandler(serverPool)

    //start load balancer
    address:=fmt.Sprintf(":%d",cfg.Port)
    log.Printf("Load Balancer listening on %s",address)
    log.Println("Ready to accept requests!")

    if err:=http.ListenAndServe(address, proxyHandler); err!=nil{
        log.Fatalf("Failed to start load balancer: %v",err)
    }
}