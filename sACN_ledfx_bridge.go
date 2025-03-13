package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Hundemeier/go-sacn/sacn"
)

type Config struct {
	Universe   int      `json:"sAcnUniverse"`
	Channel    int      `json:"channel"`
	Scenes     []string `json:"scenes"`
	LedFx_host string   `json:"ledfx_host"`
	LedFx_port string   `json:"ledfx_port"`
}

func main() {
	file, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Failed to read file: %v\n", err)
	}

	var configData Config
	err = json.Unmarshal(file, &configData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(configData)

	recv, err := sacn.NewReceiverSocket("", nil)
	if err != nil {
		log.Fatal(err)
	}
	recv.SetOnChangeCallback(func(old sacn.DataPacket, newD sacn.DataPacket) {
		fmt.Println("data changed on", newD.Universe())
	})
	recv.SetTimeoutCallback(func(univ uint16) {
		fmt.Println("timeout on", univ)
	})
	recv.Start()
	fmt.Println("start")
	select {} //only that our program does not exit. Exit with Ctrl+C
}
