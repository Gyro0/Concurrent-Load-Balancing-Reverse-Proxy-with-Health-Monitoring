package healthcheck

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"context"
	"log"
	"net/http"
	"time"
)

type HealthChecker struct {
	serverPool *servers.ServerPool
	interval time.Duration
	client *http.Client
	stopCh chan struct{}
}
func NewHealthChecker(sp *servers.ServerPool,interval string) (*HealthChecker,error){
	dur,err:=time.ParseDuration(interval)
	if err!=nil{
		return nil,err
	}
	return &HealthChecker{
		serverPool:sp,
		interval:dur,
		client:&http.Client{Timeout:5*time.Second},
		stopCh:make(chan struct{}),
	},nil
}
func (checker *HealthChecker) Start(ctx context.Context){
	ticker:=time.NewTicker(checker.interval)

	go func(){
		defer ticker.Stop()
		log.Println("[HEALTH] Starting initial health check...")
		checker.checkAll()
		for{
			select{
			case <-ctx.Done():
				log.Println("[HEALTH] Health checker stopped via context cancellation")
				return
			case <-checker.stopCh:
                log.Println("[HEALTH] Health checker stopped via Stop() call")
                return
			case <-ticker.C:
				log.Printf("[HEALTH] Running periodic health check (interval: %v)",checker.interval)
				checker.checkAll()
			}
		}
	}()

	log.Printf("[HEALTH] Health checker started (interval: %v)",checker.interval)
}

func (checker *HealthChecker) Stop() {
	close(checker.stopCh)
    time.Sleep(100*time.Millisecond) //goroutine time to exit
    log.Println("[HEALTH] Health checker shutdown complete")
}

func (checker *HealthChecker) checkAll(){
	checker.serverPool.Mux.RLock()
	backends:=make([]*backend.Backend,len(checker.serverPool.Backends))
	copy(backends,checker.serverPool.Backends)
	checker.serverPool.Mux.RUnlock()
	if len(backends)==0{
        log.Println("[HEALTH] No backends to check")
        return
    }
    log.Printf("[HEALTH] Checking %d backends...",len(backends))
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

	resp,err :=checker.client.Do(req)
	if err!=nil{
		checker.serverPool.SetBackendStatus(b.URL,false)
		return 
	}

	defer resp.Body.Close()
	alive:= resp.StatusCode>=200 && resp.StatusCode <400

	checker.serverPool.SetBackendStatus(b.URL,alive)
}
