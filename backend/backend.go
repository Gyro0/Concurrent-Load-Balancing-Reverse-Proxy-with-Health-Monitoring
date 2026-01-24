package backend

import (
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL *url.URL `json:"url"`
	Alive bool `json:"alive"`
	CurrentConns int64 `json:"current_connections"`
	mux sync.RWMutex
}


func (b *Backend) SetAlive(alive bool){
	b.mux.Lock()
	b.Alive=alive
	b.mux.Unlock()
}
func (b *Backend) IsAlive() bool{
	b.mux.Lock()
	alive:=b.Alive
	b.mux.Unlock()
	return alive
}

func (b *Backend) IncConns(){
    atomic.AddInt64(&b.CurrentConns,1)
}

func (b *Backend) DecConns(){
    atomic.AddInt64(&b.CurrentConns,-1)
}

func (b *Backend) GetConns() int64{
    return atomic.LoadInt64(&b.CurrentConns)
}