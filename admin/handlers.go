package admin

import (
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/backend"
	"Concurrent-Load-Balancing-Reverse-Proxy-with-Health-Monitoring/servers"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

//constructor to create a new admin API instance with access to the server pool 
//it provides http endpoints for runtime management of backends
func NewAdminAPI(sp *servers.ServerPool) *AdminAPI {
	return &AdminAPI{
		serverPool: sp,
	}
}

//processes GET requests
//it returns information about all backends
//info includes: health and connections
func (a *AdminAPI) HandleGet(w http.ResponseWriter, req *http.Request){

	//check if its a GET request
	if req.Method!=http.MethodGet{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}

	//getting all backends from the pool
	backends:=a.serverPool.GetAllBackends()
	var bInfo []BackendInfo
	activeCount:=0

	//iterate over backends and build info objects
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
	//constructing one complete status response
	response:=StatusResponse{
		TotalBackends:len(backends),
		ActiveBackends:activeCount,
		Backends:bInfo,
	}

	//sending JSON response
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	err:=json.NewEncoder(w).Encode(response)
	if err!=nil{
		log.Printf("Error encoding status response: %v",err)
	}
	log.Printf("[ADMIN] Status check: %d total, %d active",len(backends),activeCount)
}


//processes POST requests to add a new backend to the pool
//it validates the URL format and checks for duplicates
//returns 201 Created on success or appropriate error codes on failure
func (a *AdminAPI) HandlePost(w http.ResponseWriter, req *http.Request){
	
	//check if its a POST request
	if req.Method!=http.MethodPost{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}

	//parse the JSON request body into our request struct (AddBackendRequest)
	var r AddBackendRequest
	err:=json.NewDecoder(req.Body).Decode(&r)
	if err!=nil{
		log.Printf("[ADMIN] Invalid JSON: %v",err)
		http.Error(w,"Invalid request body",http.StatusBadRequest)
		return
	}
	//validate that a URL was provided
	if r.URL==""{
		log.Printf("[ADMIN] Empty URL provided")
		http.Error(w,"URL is required",http.StatusBadRequest)
		return
	}
	//parse the URL string to ensure it's valid
	parsedURL, err:=url.Parse(r.URL)
	if err!=nil{
		log.Printf("[ADMIN] Invalid URL format %s: %v",r.URL,err)
		http.Error(w,"Invalid URL format",http.StatusBadRequest)
		return
	}
	if parsedURL.Scheme==""||parsedURL.Host==""{
		log.Printf("[ADMIN] Incomplete URL: %s",r.URL)
		http.Error(w,"URL must be valid (eg., http://localhost:8082)",http.StatusBadRequest)
		return
	}
	//create a new backend struct
	//initially marked alive is false so that the healthchecker will check it
	newBackend:=&backend.Backend{
		URL:parsedURL,
		Alive: false,
	}

	//add the server to the pool
	if err:=a.serverPool.AddBackend(newBackend); err!=nil {
        log.Printf("[ADMIN] Failed to add backend: %v",err)
        http.Error(w,err.Error(),http.StatusConflict) //409 conflict for dups
        return
    }
	//success response
	response:=map[string]string{
		"message":"Backend added successfully",
		"url":parsedURL.String(),
	}
	//sending JSON response with 201 Created status
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("[ADMIN] Added new backend via Admin API: %s",parsedURL.String())
}

//processes DELETE requests to remove a backend from the the pool
//it validates the URL format and removes the matching backend if found
//returns 200 OK on success, 404 Not Found if backend doesn't exist
func (a *AdminAPI) HandleDelete(w http.ResponseWriter, req *http.Request){
	//check if its a DELETE request
	if req.Method!=http.MethodDelete{
		http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
		return
	}

	//parse the JSON request body into our request struct (AddBackendRequest)
	var r AddBackendRequest
	err:=json.NewDecoder(req.Body).Decode(&r)
	if err!=nil{
		log.Printf("[ADMIN] Invalid JSON: %v",err)
		http.Error(w,"Invalid request body",http.StatusBadRequest)
		return
	}

	//validate that a URL was provided
	if r.URL==""{
		log.Printf("[ADMIN] Empty URL provided: %v",err)
		http.Error(w,"URL is required",http.StatusBadRequest)
		return
	}

	//parse the URL string to ensure its valid
	parsedURL, err:=url.Parse(r.URL)
	if err!=nil{
		log.Printf("[ADMIN] Invalid URL format %s: %v",r.URL,err)
		http.Error(w,"Invalid URL format",http.StatusBadRequest)
		return
	}
	if parsedURL.Scheme==""||parsedURL.Host==""{
		log.Printf("[ADMIN] Incomplete URL: %s",r.URL)
		http.Error(w,"URL must be valid (eg., http://localhost:8082)",http.StatusBadRequest)
		return
	}
	//remove server from the pool
	removed:=a.serverPool.RemoveBackend(parsedURL)
	if removed{
		//if removed succesfully build success reponse
		response:=map[string]string{
			"message":"Backend removed successfully",
			"url":parsedURL.String(),
		}
		//sending JSON response with 200 OK status
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		log.Printf("[ADMIN] Removed backend via Admin API: %s",parsedURL.String())
	} else{
		http.Error(w,"Backend not found",http.StatusNotFound)
		log.Printf("[ADMIN] Failed to remove backend (not found): %s",parsedURL.String())
	}
}
