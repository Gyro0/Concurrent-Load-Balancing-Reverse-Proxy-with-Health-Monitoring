package healthcheck

import (
	"context"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"log"
	"net/http"
	"time"
)

type HealthChecker struct {
	serverPool *servers.ServerPool
	interval time.Duration
}
func NewHealthChecker(sp *servers.ServerPool,interval string) (*HealthChecker,error){
	dur,err:=time.ParseDuration(interval)
	if err!=nil{
		return nil,err
	}
	return &HealthChecker{
		serverPool:sp,
		interval:dur,
	},nil
}
func (checker *HealthChecker) Start(){
	ticker:=time.NewTicker(checker.interval)
	go func(){
		checker.checkAll()
		for range ticker.C {
			checker.checkAll()
		}
	}()
	log.Printf("Health checker started (interval: %v)",checker.interval)
}
func (checker *HealthChecker) checkAll(){
	checker.serverPool.Mux.RLock()
	backends:=make([]*backend.Backend,len(checker.serverPool.Backends))
	copy(backends,checker.serverPool.Backends)
	checker.serverPool.Mux.RUnlock()
	for _,b:=range backends{
		go checker.checkBackend(b)
	}
}

func (checker *HealthChecker) checkBackend(b *backend.Backend){
	ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
	defer cancel()
	req,err:= http.NewRequestWithContext(ctx,"GET",b.URL.String(),nil)
	if err!=nil{
		checker.serverPool.SetBackendStatus(b.URL,false)
		return
	}
	client :=&http.Client{Timeout:5*time.Second}
	resp,err :=client.Do(req)
	if err!=nil{
		checker.serverPool.SetBackendStatus(b.URL,false)
		return 
	}
	defer resp.Body.Close()
	alive:= resp.StatusCode>=200 && resp.StatusCode <400
	checker.serverPool.SetBackendStatus(b.URL,alive)
}

