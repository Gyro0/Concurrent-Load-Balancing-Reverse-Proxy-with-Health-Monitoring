package servers

import (
	"fmt"
	"lb/config"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type Backend struct {
	URL *url.URL `json:"url"`
	Alive bool `json:"alive"`
	CurrentConns int64 `json:"current_connections"`
	mux sync.RWMutex
}

type ServerPool struct {
	Backends []*Backend `json:"backends"`
	Current uint64 `json:"current"` // Used for Round-Robin
}

func (sp *ServerPool) LoadBackends(){
	config:=config.LoadConfig()

	for _,backendURL :=range config.Backends {
		parsedURL,err:=url.Parse(backendURL)
		if err!=nil{
			log.Fatalf("failed to parse url %s : %v ",backendURL,err)
		}
		b:=&Backend{
			URL:parsedURL,
			Alive:true,
			CurrentConns:0,
		}
		sp.Backends=append(sp.Backends,b)
	}
}

func RunServers(){
	var sp ServerPool
	sp.LoadBackends()

	var wg sync.WaitGroup
	defer wg.Wait()

	limit:=len(sp.Backends)

	for x:=0;x<limit;x++{
		wg.Add(1)
		go makeServers(&sp, &wg,x)
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