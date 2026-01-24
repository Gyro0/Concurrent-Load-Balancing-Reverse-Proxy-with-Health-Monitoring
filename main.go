package main

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/admin"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/config"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/healthcheck"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/proxy"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//This main func is the entry point of the load balancer application
//it orchestrates the initilization of all components :
//config loading, serverpool creation, health checking, proxy server and admin API
func main() {

    //define command line flag for config file path
    //usage --config=config.json / --config=pathToFile
    configFile:=flag.String("config", "config.json", "Path to configuration file")
    flag.Parse()

    //create a context that listens for interrupt signals (CTRL+C or kill)
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()


    //load configuration from the specified JSON file
    //this reads all settings including proxy port,admin port, strategy, backends, and health check frequency
    cfg:=config.LoadConfigFromFile(*configFile)
    log.Printf("[INIT] Starting Load Balancer on port %d with strategy: %s",cfg.Port,cfg.Strategy)

    //create a new server pool with specified load balancing strategy
    //this manages all backend servers and implements the load balancing logic
    serverPool:=servers.NewServerPool(cfg.Strategy)

    //load all backend server URLs from configuration into the server pool
    if err:=serverPool.LoadBackends(cfg.Backends); err!=nil {
        log.Fatalf("[WARN] Failed to load backends: %v",err)
    }

    //the following code can start test backend servers in the background

    //start backend servers in background
    //go servers.RunServers(serverPool)
    //log.Println("[INIT] Backend servers starting...")

    //initialize the health checker with the server pool and configured check frequency
    healthcheck,err:=healthcheck.NewHealthChecker(serverPool,cfg.HealthCheckFreq)
    if err!=nil{
        log.Fatalf("[WARN] Failed to start health checker: %v",err)
    }    
    log.Println("[INIT] Health Checker service starting...")

    //start the health checker in a separate goroutine
    healthcheck.Start(ctx)

    //create the proxy handler that will forward incoming requests to healthy backends
    //this is the core component that implements reverse proxy functionality
    proxyHandler:=proxy.NewProxyHandler(serverPool)
    
    //configure the HTTP server with appropriate timeouts
    //these timeouts prevent slow clients from holding connections indefinitely
    proxyServer := &http.Server{
        Addr:fmt.Sprintf(":%d",cfg.Port),
        Handler:proxyHandler,
        ReadTimeout:10*time.Second,
        WriteTimeout:10*time.Second,
        IdleTimeout:120*time.Second,
        ReadHeaderTimeout:5*time.Second,
    }

    //start the load balancer server in a goroutine so it doesn't block
    go func() {
        log.Printf("[INIT] Load Balancer listening on %s",proxyServer.Addr)
        err:=proxyServer.ListenAndServe()
        if err!=nil&& err!=http.ErrServerClosed{
            log.Fatalf("[WARN] Failed to start load balancer: %v",err)
        }
        }()
    
    
    //initialize the admin API for runtime management of backends
    //provides REST endpoints to view status and add/remove backends dynamically
    adminAPI:=admin.NewAdminAPI(serverPool)
    adminMux:=http.NewServeMux()

    //register the status endpoint (GET /status) to view all backends health and connections
    adminMux.HandleFunc("/status",adminAPI.HandleGet)

    //register the backends endpoint with method-based routing
    //POST to add new backend, DELETE to remove existing backend
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
    //configure the admin API server with timeouts
    adminServer:=&http.Server{
        Addr:fmt.Sprintf(":%d",cfg.AdminPort),
        Handler: adminMux,
        ReadTimeout: 7*time.Second,
        WriteTimeout: 7*time.Second,
    }
    //start the admin server in a separate goroutine
    //this runs independently from the proxy server
    go func(){
        log.Printf("[INIT] Admin listening on %s",adminServer.Addr)
        err:=adminServer.ListenAndServe()
        if err!=nil&& err!=http.ErrServerClosed{
            log.Fatalf("[WARN] Failed to start admin server: %v", err)
        }
    }()
    
    log.Println("[INIT] All services started successfully!")


    //block here and wait for interrupt signal (CTRL+C or kill)
    //once received the context will be cancelled and we proceed to shutdown
    <-ctx.Done()
    log.Println("[SHUTDOWN] Shutdown signal received, gracefully shutting down...")

    //create a timeout context for graceful shutdown
    //gives servers 10 seconds to finish handling existing requests
    shutdownCtx,cancel:=context.WithTimeout(context.Background(),10*time.Second)
    defer cancel()

    //stop the health checker first to prevent it from marking backends during shutdown
    healthcheck.Stop()

    //gracefully shutdown the proxy server
    //waits for existing connections to complete or timeout to expire
    if err:=proxyServer.Shutdown(shutdownCtx);err!=nil{
        log.Printf("[WARN] Proxy server shutdown error: %v",err)
    }else{
        log.Printf("[SHUTDOWN] Proxy server shutting down...")

    }

    //gracefully shutdown the admin server
    //ensures any admin requests are completed
    if err:=adminServer.Shutdown(shutdownCtx); err!=nil{
        log.Printf("[WARN] Admin server shutdown error: %v",err)
    }else{
        log.Printf("[SHUTDOWN] Admin server shutting down...")
    }
    log.Println("[SHUTDOWN] Servers gracefully stopped")
}