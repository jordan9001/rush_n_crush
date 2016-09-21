package rush_n_crush

import (
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
)

var id_counter int8 = 0
var con_read chan command

type command struct {
	client  int8
	message string
}

func readClient(ws *websocket.Conn, con_id int8) {
	data := make([]byte, 32*1024)
	for {
		// continuously read from this connection
		nr, er := ws.Read(data)
		if nr > 0 {
			message := string(data[:nr])
			fmt.Printf("Client %d : %q\n", con_id, message)
			con_read <- command{con_id, message}
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
		to_write := <-con_write
		// We have a message to deliver, write it out
		nw, ew := ws.Write(to_write)
		if ew != nil {
			fmt.Printf("Client %d Write Errored\n", con_id)
			return
		} else if nw != len(to_write) {
			fmt.Printf("Client %d Wrote Short\n", con_id)
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

	// Add this connection to our connection map
	Clients[con_id] = GameClient{
		Id:       con_id,
		ConWrite: con_write,
	}

	// Alert the Game Engine
	con_read <- command{con_id, "get_gamestate:"}

	// Begin communication loop
	go readClient(ws, con_id)
	writeClient(ws, con_write, con_id)
}

func StartServer(path, port, startup_path string) {
	var err error
	// First set up our GameLogic
	con_read, err = StartGame(startup_path)
	if err != nil {
		panic(err)
	}

	// We make our server, which will just accept any websocket connections, and pass them on to our Handler
	ws_server := websocket.Server{Handler: handleClient}

	http.Handle(path, ws_server)
	fmt.Printf("Listening on ws://localhost%s%s\n", port, path)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
