package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

type Config struct {
	IPs    []string   `json:"ips"`
	Routes [][]string `json:"routes"`
}

func GetConfiguration() (Config, error) {
	configFlag := flag.String("c", "config.json", "path to config file")
	flag.Parse()
	configPath, err := os.Open(*configFlag)
	rawConfig, err := io.ReadAll(configPath)

	if err := configPath.Close(); err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(rawConfig, &config)
	if err != nil {
		log.Printf("config file: %s\n", err.Error())
		return config, err
	}

	for _, route := range config.Routes {
		if len(route) != 2 {
			log.Printf("route length: %d\n", len(route))
			return Config{}, fmt.Errorf("route length: %d\n", len(route))
		}
	}

	return config, nil
}

func main() {

	network := NewNetwork()

	config, err := GetConfiguration()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	for _, ip := range config.IPs {
		err := network.AddRouter(ip)
		if err != nil {
			log.Print(err)
			os.Exit(2)
		}
	}

	for _, route := range config.Routes {
		err := network.AddEdge(route[0], route[1])
		if err != nil {
			log.Print(err)
			os.Exit(3)
		}
	}

	network.Run()
}
