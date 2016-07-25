package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

var cfg *config

type config struct {
	KeybasePath string `json:"Keybase_path"`
	TCPHost     string `json:"tcp_host"`
	HTTPHost    string `json:"http_host"`
}

func loadConfig() {
	f, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	var c config
	err = json.Unmarshal(f, &c)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Contains(c.KeybasePath, "/") || strings.Contains(c.KeybasePath, "\\") {
		p, err := filepath.Abs(c.KeybasePath)
		if err != nil {
			log.Fatal(err)
		}
		c.KeybasePath = p
	}
	cfg = &c
}

func main() {
	loadConfig()
	c := &client{
		httpClient: &http.Client{},
	}
	c.connect()
	c.readInput()
}
