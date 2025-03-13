package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Hundemeier/go-sacn/sacn"
)

type Config struct {
	Universe   int      `json:"sAcnUniverse"`
	Channel    int      `json:"channel"`
	Scenes     []string `json:"scenes"`
	LedFx_host string   `json:"ledfx_host"`
	LedFx_port string   `json:"ledfx_port"`
}

func activateScene(sceneId string, deactivate bool) {
	var action = "activate"
	if deactivate {
		ActiveScene = "OFF"
		action = "deactivate"
	} else {
		ActiveScene = sceneId
	}
	fmt.Println(action, sceneId)

	payload := map[string]interface{}{"id": sceneId, "action": action}
	out, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		"http://"+configData.LedFx_host+":"+configData.LedFx_port+"/api/scenes",
		strings.NewReader(string(out)),
	)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	_, err = client.Do(req)
	if err != nil {
		// handle error
		log.Fatal(err)
	}

}

var ActiveScene = "OFF"

var configData Config

func main() {
	file, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Failed to read file: %v\n", err)
	}

	err = json.Unmarshal(file, &configData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(configData)

	recv, err := sacn.NewReceiverSocket("", nil)
	if err != nil {
		log.Fatal(err)
	}
	var lastChannelValue byte = 0

	recv.SetOnChangeCallback(func(old sacn.DataPacket, newD sacn.DataPacket) {
		fmt.Println("data changed on", newD.Universe())

		var channelValue = newD.Data()[configData.Channel-1]
		fmt.Println("selected Channel value: ", channelValue)
		if channelValue != lastChannelValue {
			lastChannelValue = channelValue

			if channelValue == 0 {
				activateScene(ActiveScene, true)
			} else {
				if channelValue <= byte(len(configData.Scenes)) {
					activateScene(configData.Scenes[channelValue-1], false)
				}
			}
		}
	})
	recv.SetTimeoutCallback(func(univ uint16) {
		fmt.Println("timeout on", univ)
	})
	recv.Start()
	fmt.Println("start")
	select {} //only that our program does not exit. Exit with Ctrl+C
}
