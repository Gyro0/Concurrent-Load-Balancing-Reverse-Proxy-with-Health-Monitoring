package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run server.go <port>")
        fmt.Println("Example: go run server.go 8082")
        os.Exit(1)
	}
	
	port:=os.Args[1]
	name:=fmt.Sprintf("Backend-%s",port)
	mux:=http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter,req *http.Request){
		log.Printf("[%s] Received request: %s %s from %s",name,req.Method,req.URL.Path,req.RemoteAddr)

		select {
        case <-time.After(1 * time.Second): //simulate work
			w.WriteHeader(http.StatusOK)

            response:=fmt.Sprintf("Response from %s\nTime: %s\nPath: %s\n", 
                name, 
                time.Now().Format(time.RFC3339), 
                req.URL.Path)
            fmt.Fprint(w, response)
			log.Printf("[%s] Response sent successfully",name)


        case <-req.Context().Done():
            //client disconnected = stop processing
            log.Printf("[INFO] Client disconnected while processing request on %s",name)
            return
        }
	})
	address:=fmt.Sprintf(":%s",port)
    log.Printf("[%s] Starting server on http://localhost%s",name,address)
    
    if err:=http.ListenAndServe(address, mux); err!=nil{
        log.Fatalf("[%s] Server failed: %v",name,err)
    }
}
