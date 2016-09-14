package rush_n_crush

import (
	"fmt"
)

type GameClient struct {
	Id       int8
	ConWrite chan string
	ConRead  chan string
}

var Clients map[int8]GameClient

func GameLoop() {
	for {
		var message string = ""
		// Read from our clients
		for id, v := range Clients {
			select {
			case msg := <-v.ConRead:
				message = msg
			default:
				message = message
			}
		}
		// Write out to all our clients
		if len(message) > 0 {
			for id, v := range Clients {
				v.ConWrite <- message
			}
		}
	}
}

func StartGame() {
	// Make our client map
	Clients = make(map[int8]GameClient)
	go GameLoop()
}
