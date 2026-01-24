package admin

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

func NewAdminAPI(sp *servers.ServerPool) *AdminAPI {
	return &AdminAPI{
		serverPool: sp,
	}
}

// GET /status
func (a *AdminAPI) HandleGet(w http.ResponseWriter, req *http.Request){
	if req.Method!=http.MethodGet{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}
	backends:=a.serverPool.GetAllBackends()
	var bInfo []BackendInfo
	activeCount:=0
	for _,b:=range backends{
		info:=BackendInfo{
			URL:b.URL.String(),
			Alive:b.IsAlive(),
			CurrentConns:b.GetConns(),
		}
		bInfo=append(bInfo,info)
		if b.IsAlive(){
			activeCount++
		}
	}
	response:=StatusResponse{
		TotalBackends:len(backends),
		ActiveBackends:activeCount,
		Backends:bInfo,
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	err:=json.NewEncoder(w).Encode(response)
	if err!=nil{
		log.Printf("Error encoding status response: %v",err)
	}
	log.Printf("Status check: %d total, %d active",len(backends),activeCount)
}


//POST /backends
func (a *AdminAPI) HandlePost(w http.ResponseWriter, req *http.Request){
	if req.Method!=http.MethodPost{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}

	var r AddBackendRequest
	err:=json.NewDecoder(req.Body).Decode(&r)
	if err!=nil{
		http.Error(w,"Invalid request body",http.StatusBadRequest)
		return
	}
	if r.URL==""{
		http.Error(w,"URL is required",http.StatusBadRequest)
		return
	}
	parsedURL, err:=url.Parse(r.URL)
	if err!=nil{
		http.Error(w,"Invalid URL format",http.StatusBadRequest)
		return
	}
	if parsedURL.Scheme==""||parsedURL.Host==""{
		http.Error(w,"URL must be valid (eg., http://localhost:8082)",http.StatusBadRequest)
		return
	}
	newBackend:=&backend.Backend{
		URL:parsedURL,
		Alive: true,
	}
	a.serverPool.AddBackend(newBackend)
	response:=map[string]string{
		"message":"Backend added successfully",
		"url":parsedURL.String(),
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Added new backend via Admin API: %s",parsedURL.String())
}

//DELETE /backends
func (a *AdminAPI) HandleDelete(w http.ResponseWriter, req *http.Request){
	if req.Method!=http.MethodDelete{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}

	var r AddBackendRequest
	err:=json.NewDecoder(req.Body).Decode(&r)
	if err!=nil{
		http.Error(w,"Invalid request body",http.StatusBadRequest)
		return
	}
	if r.URL==""{
		http.Error(w,"URL is required",http.StatusBadRequest)
		return
	}
	parsedURL, err:=url.Parse(r.URL)
	if err!=nil{
		http.Error(w,"Invalid URL format",http.StatusBadRequest)
		return
	}
	if parsedURL.Scheme==""||parsedURL.Host==""{
		http.Error(w,"URL must be valid (eg., http://localhost:8082)",http.StatusBadRequest)
		return
	}
	removed:=a.serverPool.RemoveBackend(parsedURL)
	if removed{
		response:=map[string]string{
			"message":"Backend removed successfully",
			"url":parsedURL.String(),
		}
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		log.Printf("Removed backend via Admin API: %s",parsedURL.String())
	} else{
		http.Error(w,"Backend not found",http.StatusNotFound)
		log.Printf("Failed to remove backend (not found): %s",parsedURL.String())
	}
}
