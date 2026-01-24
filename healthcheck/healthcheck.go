package healthcheck

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"context"
	"log"
	"net/http"
	"time"
)

//HealthChecker periodically monitors the health of backend servers
//it sends HTTP requests to each backend and updates their status
//this ensures that the load balancer only forwards requests to activa servers
type HealthChecker struct {
	serverPool *servers.ServerPool
	interval time.Duration
	client *http.Client
	stopCh chan struct{}
}

//constructor that initializes a new healthchecker instance
//parses the interval string and configures the http client for timeout
func NewHealthChecker(sp *servers.ServerPool,interval string) (*HealthChecker,error){
	dur,err:=time.ParseDuration(interval)
	if err!=nil{
		return nil,err
	}
	return &HealthChecker{
		serverPool:sp,
		interval:dur,
		client:&http.Client{Timeout:5*time.Second}, //5sec timeout for each healthcheck
		stopCh:make(chan struct{}), //unbuffered channel for stop signal
	},nil
}

//this func begins the periodic health checking routine
//runs a separate goroutine and can be stopped via context or stop() func
func (checker *HealthChecker) Start(ctx context.Context){
	//ticker that fires at the configurated interval
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
			case <-ticker.C: //time to perform another health check 
				log.Printf("[HEALTH] Running periodic health check (interval: %v)",checker.interval)
				checker.checkAll()
			}
		}
	}()

	log.Printf("[HEALTH] Health checker started (interval: %v)",checker.interval)
}

//gracefully shuts down the healthchecker
//it closes the stop channel
func (checker *HealthChecker) Stop() {
	close(checker.stopCh)
    time.Sleep(100*time.Millisecond) //goroutine time to exit
    log.Println("[HEALTH] Health checker shutdown complete")
}

//it retrieves all backends from the server pool and check each one concurrently
func (checker *HealthChecker) checkAll(){
	//acquire read lock to safely copy the backends
	checker.serverPool.Mux.RLock()
	backends:=make([]*backend.Backend,len(checker.serverPool.Backends))
	copy(backends,checker.serverPool.Backends)
	checker.serverPool.Mux.RUnlock()
	//skip the check if no backends exist
	if len(backends)==0{
        log.Println("[HEALTH] No backends to check")
        return
    }
    log.Printf("[HEALTH] Checking %d backends...",len(backends))

	//spawn a goroutine for each backend to check them all simultaneously
    //this prevents slow backends from delaying checks of healthy ones
	for _,b:=range backends{
		go checker.checkBackend(b)
	}
}


//this performs a health check on one backends server
//it sends an HTTP GET request and considers it healthy if it responds with 2xx or 3xx code status
//and updates the status based on the check result
func (checker *HealthChecker) checkBackend(b *backend.Backend){
	//context with timeout to prevent handing on unresponsive backends
	ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second)
	defer cancel()

	//create an http get request with the timeout context
	req,err:= http.NewRequestWithContext(ctx,"GET",b.URL.String(),nil)

	if err!=nil{
		//if we cant even create a request mark the backend as down
		checker.serverPool.SetBackendStatus(b.URL,false)
		return
	}
	//execute the request 
	resp,err :=checker.client.Do(req)
	if err!=nil{
		//any error means the backend is down
		checker.serverPool.SetBackendStatus(b.URL,false)
		return 
	}

	defer resp.Body.Close()
	//consider the backend alive if it returns a successful status code
	alive:= resp.StatusCode>=200 && resp.StatusCode <400
	//updates the status in the serverpool
	checker.serverPool.SetBackendStatus(b.URL,alive)
}
