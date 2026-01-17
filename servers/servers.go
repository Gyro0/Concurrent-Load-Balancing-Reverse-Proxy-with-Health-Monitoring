package servers

import (
	"fmt"
	"lb/backend"
	"lb/config"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
)



type ServerPool struct {
	Backends []*backend.Backend `json:"backends"`
	Current uint64 `json:"current"` // Used for Round-Robin
	mux sync.RWMutex
}

func NewServerPool() *ServerPool {
	return &ServerPool{
		Backends:make([]*backend.Backend,0),
	}
}

func (sp *ServerPool) LoadBackends(){
	config:=config.LoadConfig()

	for _,backendURL :=range config.Backends {
		parsedURL,err:=url.Parse(backendURL)
		if err!=nil{
			log.Fatalf("failed to parse url %s : %v ",backendURL,err)
		}
		b:=&backend.Backend{
			URL:parsedURL,
			Alive:true,
			CurrentConns:0,
		}
		sp.AddBackend(b)
	}
}

func (sp *ServerPool) AddBackend(b *backend.Backend) {
	sp.mux.Lock()
	defer sp.mux.Unlock()
	sp.Backends=append(sp.Backends,b)
	log.Printf("Added backend: %s",b.URL)
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
		fmt.Fprintf(w, "Backend server at %s\n", backend.URL.String())
	})
	address:=fmt.Sprintf(":%s",backend.URL.Port())	
	log.Printf("Starting backend server on %s",address)
	if err:=http.ListenAndServe(address, r); err!=nil {
        log.Printf("Server error on %s: %v", address, err)
    }
}


func (sp *ServerPool) getNextValidPeerRR() *backend.Backend{
	sp.mux.RLock()
	defer sp.mux.RUnlock()
	len:=len(sp.Backends)
	if len==0{
		return nil
	}
	start:=atomic.AddUint64(&sp.Current,1)% uint64(len)
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
	sp.mux.RLock()
	defer sp.mux.RUnlock()
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
	cfg := config.LoadConfig()

	switch cfg.Strategy{
		case "round-robin","rr":
			return sp.getNextValidPeerRR()
		case "least-connections","lc":
			return sp.getNextValidPeerLC()
		default:
			log.Printf("unknown strategy '%s', default is rr",cfg.Strategy)
			return sp.getNextValidPeerRR()
	}
}

func (sp *ServerPool) SetBackendStatus(link *url.URL, alive bool){
	sp.mux.RLock()
	defer sp.mux.RUnlock()
	for _,b :=range sp.Backends{
		if b.URL.String()==link.String(){
			b.SetAlive(alive)
			if !alive{
				log.Printf("Backend %s marked as DOWN",link)
			}else{
				log.Printf("Backend %s marked as UP",link)
			}
			break
		}
	}
}
