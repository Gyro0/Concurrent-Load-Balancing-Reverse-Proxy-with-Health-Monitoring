package servers

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)



type ServerPool struct {
	Backends []*backend.Backend `json:"backends"`
	Current uint64 `json:"current"` // Used for Round-Robin
	Mux sync.RWMutex
	strategy string
}

func NewServerPool(strategy string) *ServerPool {
	return &ServerPool{
		Backends:make([]*backend.Backend,0),
		strategy:strategy,
	}
}

func (sp *ServerPool) LoadBackends(urls []string)error{
	for _,backendURL :=range urls {
		parsedURL,err:=url.Parse(backendURL)
		if err!=nil{
			log.Fatalf("failed to parse url %s : %v ",backendURL,err)
			return err
		}
		b:=&backend.Backend{
			URL:parsedURL,
			Alive:true,
			CurrentConns:0,
		}
		sp.AddBackend(b)
	}
	return nil
}

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

//servers control

func RunServers(sp *ServerPool){
	var wg sync.WaitGroup
	defer wg.Wait()

	limit:=len(sp.Backends)
	for x:=0;x<limit;x++{
		wg.Add(1)
		go makeServers(sp, &wg,x)
	}
}
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


func (sp *ServerPool) getNextValidPeerRR() *backend.Backend{
	sp.Mux.RLock()
	defer sp.Mux.RUnlock()
	len:=len(sp.Backends)
	if len==0{
		return nil
	}
	start:=(atomic.AddUint64(&sp.Current,1)-1)% uint64(len)
	for i:=0;i<len;i++{
		idx:=(start+uint64(i))%uint64(len)
		b:=sp.Backends[idx]
		if b.IsAlive(){
			return b
		}
	}
	return nil
}

func (sp *ServerPool) getNextValidPeerLC() *backend.Backend{
	sp.Mux.RLock()
	defer sp.Mux.RUnlock()
	if len(sp.Backends)==0{
		return nil
	}
	var leastConnsBackend *backend.Backend
	minConns :=int64(-1)
	for _,b:=range sp.Backends{
		if !b.IsAlive(){
			continue
		}
		conns:=b.GetConns()
		if minConns==-1 || conns<minConns{
			minConns=conns
			leastConnsBackend=b
		}
	}
	return leastConnsBackend
}

func (sp *ServerPool) GetNextValidPeer() *backend.Backend{
	switch sp.strategy{
		case "round-robin","rr","Round-Robin","Round-robin","RR":
			return sp.getNextValidPeerRR()
		case "least-connections","lc","Least-Connections","Least-connections","LC":
			return sp.getNextValidPeerLC()
		default:
			log.Printf("unknown strategy '%s', default is rr",sp.strategy)
			return sp.getNextValidPeerRR()
	}
}

func (sp *ServerPool) SetBackendStatus(link *url.URL, alive bool){
	backends:=sp.GetAllBackends()
	for _,b :=range backends{
		if b.URL.String()==link.String(){
			wasAlive:=b.IsAlive()
            b.SetAlive(alive)
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
	log.Printf("[WARN] Attempted to set status for unknown backend: %s", link)

}


func (sp *ServerPool) GetAllBackends() []*backend.Backend{
	sp.Mux.RLock()
	backends:=make([]*backend.Backend,len(sp.Backends))
	copy(backends,sp.Backends)
	sp.Mux.RUnlock()
	return backends
}

func (sp *ServerPool) RemoveBackend(u *url.URL) bool{
	sp.Mux.Lock()
	defer sp.Mux.Unlock()
	for i,b :=range sp.Backends{
		if b.URL.String()==u.String(){
			sp.Backends=append(sp.Backends[:i],sp.Backends[i+1:]...)
			return true
		}
	}
	return false
}