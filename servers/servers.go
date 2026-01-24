package servers

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

//a server pool manages a collection of backend server and implements loadbalancing algorithms
//it maintains the list of backends and their health status and provides thread safe access
//it implements the LoadBalancer interface to allow different selection strategies
type ServerPool struct {
	Backends []*backend.Backend `json:"backends"`
	Current uint64 `json:"current"` // Used for Round-Robin
	Mux sync.RWMutex
	strategy string
}

//constructor that initializes a new empty server pool with the specified load balancing strategy
//the strategy is what determines which algorithm we are gonna use to select backends (round-roubin or least-connections)
func NewServerPool(strategy string) *ServerPool {
	return &ServerPool{
		Backends:make([]*backend.Backend,0),
		strategy:strategy,
	}
}


//this func parses a list of URL strings and adds them as backends to the pool 
//its used to add servers to the pool from the list declared in the json config file
func (sp *ServerPool) LoadBackends(urls []string)error{
	for _,backendURL :=range urls {
		parsedURL,err:=url.Parse(backendURL)
		if err!=nil{
			log.Fatalf("failed to parse url %s : %v ",backendURL,err)
			return err
		}

		//new backend struct with the parsed URL
		//initially mark as alive, healthchecker will verify 
		b:=&backend.Backend{
			URL:parsedURL,
			Alive:true,
			CurrentConns:0,
		}
		sp.AddBackend(b)
	}
	return nil
}

//this func adds a new backend server to the pool after checking for duplcates
//it also acquires write lock to safely modfy the backend list
//returns an error if a backend with the same url already exists in the pool
func (sp *ServerPool) AddBackend(b *backend.Backend) error {
    sp.Mux.Lock()
    defer sp.Mux.Unlock()
    //check if backend already exists
    for _,existing:=range sp.Backends{
        if existing.URL.String()==b.URL.String() {
            return fmt.Errorf("backend %s already exists",b.URL)
        }
    }
    sp.Backends=append(sp.Backends,b)
    log.Printf("[INIT] Added backend: %s",b.URL)
    return nil
}

//this is a utility function to start test backend servers locally
//its optional and used for testing, you can run independent server using server.go
func RunServers(sp *ServerPool){
	var wg sync.WaitGroup
	defer wg.Wait()

	limit:=len(sp.Backends)
	for x:=0;x<limit;x++{
		wg.Add(1)
		go makeServers(sp, &wg,x)
	}
}

//this is also a utility function that creates and runs a simple HTTP server for testing
//it responds to all requests after a brief delay to simulate work
func makeServers(sp *ServerPool, wg *sync.WaitGroup,idx int){
	defer wg.Done()

	backend:=sp.Backends[idx]
	r:= http.NewServeMux()

	r.HandleFunc("/", func(w http.ResponseWriter,req *http.Request){
		select {
        case <-time.After(100 * time.Millisecond): //simulate work
            fmt.Fprintf(w, "Response from %s\n",backend.URL.String())

        case <-req.Context().Done():
            //client disconnected = stop processing
            log.Printf("[INFO] Client disconnected while processing request on %s",backend.URL)
            return
        }
	})
	address:=fmt.Sprintf(":%s",backend.URL.Port())	
	log.Printf("[BACKEND] Starting backend server on %s",address)
	if err:=http.ListenAndServe(address, r); err!=nil {
        log.Printf("[BACKEND] Backend server %s failed: %v", address, err)
    }
}


//this func implements round-robin backend selection
//it cycles through backend in order while skipping unhealthy ones 
//it uses atomic operation on the counter to ensure thread safety
func (sp *ServerPool) getNextValidPeerRR() *backend.Backend{
	sp.Mux.RLock()
	defer sp.Mux.RUnlock()
	//number of backends in the pool
	len:=len(sp.Backends)
	if len==0{
		return nil
	}
	//atomically increment and get the current position using modulo
	start:=(atomic.AddUint64(&sp.Current,1)-1)% uint64(len)
	//try each backend in order starting from the current position
	for i:=0;i<len;i++{
		idx:=(start+uint64(i))%uint64(len)
		b:=sp.Backends[idx]
		//once we find a healthy backend we return it
		if b.IsAlive(){
			return b
		}
	}
	//if no backends found
	return nil
}


//this func implements least-connections backend selection
//it finds the healthy backend with fewer active connections
//and chooses randomly if there exist multiple min connections backends
func (sp *ServerPool) getNextValidPeerLC() *backend.Backend{
	sp.Mux.RLock()
	defer sp.Mux.RUnlock()
	if len(sp.Backends)==0{
		return nil
	}
    var candidates []*backend.Backend
	minConns :=int64(-1)
	//find all backends with minimum connection count
	for _,b:=range sp.Backends{
		if !b.IsAlive(){
			continue
		}
		conns:=b.GetConns()
		if minConns==-1 || conns<minConns {
			//found a new minimum we start a new list
            minConns=conns
            candidates=[]*backend.Backend{b}
        } else if conns==minConns {
            //same min we add to the list
            candidates = append(candidates, b)
        }
	}
	if len(candidates)==0{
        return nil
    }
    //if multiple backneds exists, choose randomly
    if len(candidates)>1{
        idx:=rand.Intn(len(candidates))
        return candidates[idx]
    }
	//return the only one
    return candidates[0]
}


//this func selects the next backend using the configured strategy, it routes to either
//round-robin or least-connections based on the serverpool's configuration
//defaults to round robin if an unknown strategy is choosen
func (sp *ServerPool) GetNextValidPeer() *backend.Backend{
	switch sp.strategy{
		case "round-robin","rr","Round-Robin","Round-robin","RR":
			return sp.getNextValidPeerRR()
		case "least-connections","lc","Least-Connections","Least-connections","LC":
			return sp.getNextValidPeerLC()
		default:
			log.Printf("[WARN] Unknown strategy '%s', default is Round-Robin",sp.strategy)
			return sp.getNextValidPeerRR()
	}
}


//this func updates the status of a backend 
//its used by the healthchecker
func (sp *ServerPool) SetBackendStatus(link *url.URL, alive bool){
	//gets a copy of all backends
	backends:=sp.GetAllBackends()
	//finds the matching backend
	for _,b :=range backends{
		if b.URL.String()==link.String(){
			wasAlive:=b.IsAlive()
            b.SetAlive(alive)
			//log only if the status actually changed
            if wasAlive!=alive {
                if alive{
                    log.Printf("[STATUS] Backend %s marked as UP",link)
                }else{
                    log.Printf("[STATUS] Backend %s marked as DOWN",link)
                }
            }
            return
		}
	}
	//if we get here no matching server was found in the pool
	log.Printf("[WARN] Attempted to set status for unknown backend: %s", link)

}

//this func returns a copy of all backends in the pool
//it creates a new list and copies the backend pointers to avoid holding the lock
func (sp *ServerPool) GetAllBackends() []*backend.Backend{
	sp.Mux.RLock()
	backends:=make([]*backend.Backend,len(sp.Backends))
	copy(backends,sp.Backends)
	sp.Mux.RUnlock()
	return backends
}

//this func removes a backend from the pool
//it acquires write lock to safely modify the backend list
//returns true if the backend was found and removed and false otherwise
func (sp *ServerPool) RemoveBackend(u *url.URL) bool{
	sp.Mux.Lock()
	defer sp.Mux.Unlock()
	
	//search for the backend with matching URL
	for i,b :=range sp.Backends{
		if b.URL.String()==u.String(){
			//this creates a new slice without the removed backend
			sp.Backends=append(sp.Backends[:i],sp.Backends[i+1:]...)
			return true
		}
	}
	return false
}