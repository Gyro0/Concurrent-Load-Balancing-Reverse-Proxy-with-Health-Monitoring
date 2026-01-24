package config
import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
    Port                  int      `json:"port"`
    Strategy              string   `json:"strategy"`
    Backends              []string `json:"backends"`
	HealthCheckFreq 	  string   `json:"health_check_frequency"`
	AdminPort			  int       `json:"admin_port"`
}

func LoadConfigFromFile(filename string) Config {
	file,err :=os.Open(filename)
	if err!=nil{
		log.Fatalf("failed to open config file %s: %v",filename,err)
	}
	defer file.Close()
	var config Config
	
    if err:=json.NewDecoder(file).Decode(&config); err!=nil{
        log.Fatalf("failed to decode config file %s: %v",filename,err)
    }
	return config
}