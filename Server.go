package rush_n_crush

import (
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
)

var id_counter int8 = 0

func readClient(ws *websocket.Conn, con_read chan string, con_id int8) {
	data := make([]byte, 32*1024)
	for {
		// continuously read from this connection
		nr, er := ws.Read(data)
		if nr > 0 {
			message := string(data[:nr])
			fmt.Printf("Client %d : %q\n", con_id, message)
			con_read <- message
		}
		if er == io.EOF {
			fmt.Printf("Client %d Disconnected\n", con_id)
			return
		} else if er != nil {
			fmt.Printf("Client %d Read Errored\n", con_id)
			return
		}
	}
}

func writeClient(ws *websocket.Conn, con_write chan []byte, con_id int8) {
	for {
		// write if we have a message to write
		select {
		case to_write := <-con_write:
			// We have a message to deliver, write it out
			nw, ew := ws.Write(to_write)
			if ew != nil {
				fmt.Printf("Client %d Write Errored\n")
				return
			} else if nw != len(to_write) {
				fmt.Printf("Client %d Wrote Short\n")
			}
		default:
			// Nothing to write, loop back
			continue
		}
	}
}

func handleClient(ws *websocket.Conn) {
	var con_id int8
	con_id = id_counter
	id_counter += 1

	fmt.Printf("Got a new client: %d\n", con_id)

	// Add channels for output and input to this connection
	con_write := make(chan []byte)
	con_read := make(chan string)

	// Add this connection to our connection map
	Clients[con_id] = GameClient{
		Id:       con_id,
		ConWrite: con_write,
		ConRead:  con_read,
	}

	// Alert the Game Engine
	con_read <- "get_gamestate:"

	// Begin communication loop
	go readClient(ws, con_read, con_id)
	writeClient(ws, con_write, con_id)
}

func StartServer(path string, port uint16) {
	portstring := fmt.Sprintf(":%d", port)
	// First set up our GameLogic
	StartGame()

	// We make our server, which will just accept any websocket connections, and pass them on to our Handler
	ws_server := websocket.Server{Handler: handleClient}

	http.Handle(path, ws_server)
	fmt.Printf("Listening on ws://localhost%s%s\n", portstring, path)
	err := http.ListenAndServe(portstring, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
