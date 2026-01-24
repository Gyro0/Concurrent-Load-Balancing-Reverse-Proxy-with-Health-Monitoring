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

func main() {
    configFile:=flag.String("config", "config.json", "Path to configuration file")
    flag.Parse()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()


    //load configuration
    cfg:=config.LoadConfigFromFile(*configFile)
    log.Printf("[INIT] Starting Load Balancer on port %d with strategy: %s",cfg.Port,cfg.Strategy)

    //create server pool and load backends
    serverPool:=servers.NewServerPool(cfg.Strategy)

    if err:=serverPool.LoadBackends(cfg.Backends); err!=nil {
        log.Fatalf("[WARN] Failed to load backends: %v",err)
    }
    //start backend servers in background
    //go servers.RunServers(serverPool)
    //log.Println("[INIT] Backend servers starting...")

    //start health checker routine
    healthcheck,err:=healthcheck.NewHealthChecker(serverPool,cfg.HealthCheckFreq)
    if err!=nil{
        log.Fatalf("[WARN] Failed to start health checker: %v",err)
    }    
    log.Println("[INIT] Health Checker service starting...")

    healthcheck.Start(ctx)

    //create proxy handler
    proxyHandler:=proxy.NewProxyHandler(serverPool)
    
    //start load balancer
    proxyServer := &http.Server{
        Addr:fmt.Sprintf(":%d",cfg.Port),
        Handler:proxyHandler,
        ReadTimeout:10*time.Second,
        WriteTimeout:10*time.Second,
        IdleTimeout:120*time.Second,
        ReadHeaderTimeout:5*time.Second,
    }

    
    go func() {
        log.Printf("[INIT] Load Balancer listening on %s",proxyServer.Addr)
        err:=proxyServer.ListenAndServe()
        if err!=nil&& err!=http.ErrServerClosed{
            log.Fatalf("[WARN] Failed to start load balancer: %v",err)
        }
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
        log.Printf("[INIT] Admin listening on %s",adminServer.Addr)
        err:=adminServer.ListenAndServe()
        if err!=nil&& err!=http.ErrServerClosed{
            log.Fatalf("[WARN] Failed to start admin server: %v", err)
        }
    }()
    
    log.Println("[INIT] All services started successfully!")

    //wait for interrupt signal
    <-ctx.Done()
    log.Println("[SHUTDOWN] Shutdown signal received, gracefully shutting down...")

    shutdownCtx,cancel:=context.WithTimeout(context.Background(),10*time.Second)
    defer cancel()
    healthcheck.Stop()

    //shutdown servers
    if err:=proxyServer.Shutdown(shutdownCtx);err!=nil{
        log.Printf("[WARN] Proxy server shutdown error: %v",err)
    }else{
        log.Printf("[SHUTDOWN] Proxy server shutting down...")

    }
    if err:=adminServer.Shutdown(shutdownCtx); err!=nil{
        log.Printf("[WARN] Admin server shutdown error: %v",err)
    }else{
        log.Printf("[SHUTDOWN] Admin server shutting down...")
    }
    log.Println("[SHUTDOWN] Servers gracefully stopped")
}