package config
import (
	"encoding/json"
	"log"
	"os"
)

//Config struct holds all configuration settings for the loadbalancer
type Config struct {
    Port                  int      `json:"port"`
    Strategy              string   `json:"strategy"`
    Backends              []string `json:"backends"`
	HealthCheckFreq 	  string   `json:"health_check_frequency"`
	AdminPort			  int       `json:"admin_port"`
}

//this func reads and parses a json config file
func LoadConfigFromFile(filename string) Config {
	file,err :=os.Open(filename)
	if err!=nil{
		log.Fatalf("failed to open config file %s: %v",filename,err)
	}
	defer file.Close()
	var config Config
	//decodes the json file contect into the config struct
    if err:=json.NewDecoder(file).Decode(&config); err!=nil{
        log.Fatalf("failed to decode config file %s: %v",filename,err)
    }
	//returns the populated config struct to use
	return config
}