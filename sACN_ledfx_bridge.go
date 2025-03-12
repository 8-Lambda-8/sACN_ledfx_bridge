package main

import (
	"fmt"
	"log"

	"github.com/Hundemeier/go-sacn/sacn"
)

func main() {
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
