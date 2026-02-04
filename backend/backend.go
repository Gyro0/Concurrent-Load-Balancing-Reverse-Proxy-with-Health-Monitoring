package backend

import (
	"net/url"
	"sync"
	"sync/atomic"
)

//struct that represents a single backend server in the server pool
//tracks URL,health status,connection count
type Backend struct {
	URL *url.URL `json:"url"`
	Alive bool `json:"alive"`
	CurrentConns int64 `json:"current_connections"`
	mux sync.RWMutex
}

//updated the server's status in a thread safe manner
func (b *Backend) SetAlive(alive bool){
	b.mux.Lock()
	b.Alive=alive
	b.mux.Unlock()
}

//returns the current status of the server
func (b *Backend) IsAlive() bool{
	b.mux.RLock()
	alive:=b.Alive
	b.mux.RUnlock()
	return alive
}

//atomically increments the connection counter by 1
//called when a new request is forwarded to this backend
func (b *Backend) IncConns(){
    atomic.AddInt64(&b.CurrentConns,1)
}

//atomically decrements the connection counter by 1
//called when a request completes and the connection is released
func (b *Backend) DecConns(){
    atomic.AddInt64(&b.CurrentConns,-1)
}

//returns the current number of active conns
func (b *Backend) GetConns() int64{
    return atomic.LoadInt64(&b.CurrentConns)
}