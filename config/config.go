package config
import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
    Port                  int      `json:"port"`
    Strategy              string   `json:"Strategy"`
    Backends              []string `json:"backends"`
	HealthCheckFreq 	  string   `json:"health_check_frequency"`
	AdminPort			  int       `json:"admin_port"`
}

func LoadConfig() Config {
	file,err :=os.ReadFile("config.json")
	if err!=nil{
		log.Fatalf("failed to read : %v",err)
	}
	var config Config
    if err := json.Unmarshal(file, &config); err != nil {
        log.Fatalf("failed to parse config file: %v", err)
    }
	return config
}