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


func (b *Backend) SetAlive(alive bool) (*Backend){
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive=alive
	return b
}
func (b *Backend) IsAlive() bool{
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.Alive
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