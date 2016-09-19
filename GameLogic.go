package rush_n_crush

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	STARTUP_FILE string = "./run/startup.cmd"
)

type GameClient struct {
	Id       int8
	ConWrite chan []byte
	ConRead  chan string
	Nick     string
}

type UpdateGroup struct {
	YourTurn      bool
	ClientTurn    string
	TileUpdates   []Tile
	PlayerUpdates []Player
}

func (u UpdateGroup) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"your_turn\":")
	if u.YourTurn {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
	buf.WriteString(",\"current_turn\":\"")
	buf.WriteString(u.ClientTurn)
	buf.WriteString("\",\"updated_tiles\":[")
	for _, v := range u.TileUpdates {
		tileJSON, _ := v.MarshalJSON()
		buf.Write(tileJSON)
		buf.WriteString(",")
	}
	buf.WriteString("],\"updated_players\":[")
	for _, v := range u.PlayerUpdates {
		playerJSON, _ := v.MarshalJSON()
		buf.Write(playerJSON)
		buf.WriteString(",")
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

// Game State Variables
var Clients map[int8]GameClient
var ClientTurn int8 = -1
var settings_playersPerClient int8 = 1

// The syntax is command:arg_string
func processCommand(id int8, message string) error {
	var err error = nil
	// preallocate the update group
	var u UpdateGroup
	u.TileUpdates = make([]Tile, 0, 16)
	u.PlayerUpdates = make([]Player, 0, 2)

	i := strings.Index(message, ":")
	if i < 0 {
		return errors.New("Got a bad command")
	}

	command := message[:i]

	// Run Commands
	switch command {
	case "get_gamestate":
		// They need to be sent the map
		SendMap(id)
		// After sending, add in players for them
		AddPlayers(id, &u)
	case "set_nick":
		// Set this client's nickname
		c := Clients[id]
		c.Nick = cleanString(message[i+1:])
		return nil
	case "set_default_moves":
		if id == -1 {
			var desired int64
			desired, err = strconv.ParseInt(message[i+1:], 10, 8)
			if err != nil {
				return err
			}
			if desired > 0 {
				movesPerPlayer = int8(desired)
			}
		}
		return nil
	case "map":
		// if a game has not been started, load a map
		if ClientTurn < 0 {
			LoadMap(message[i+1:])
		}
	case "setting_players_per_client":
		if id == -1 {
			var desired int64
			desired, err = strconv.ParseInt(message[i+1:], 10, 8)
			if err != nil {
				return err
			}
			if desired > 0 {
				settings_playersPerClient = int8(desired)
			}
		}
		return nil
	case "end_turn":
		// Move to next Client
		clearClientMoves(id, &u)
	}
	// check if we should update state (who's turn it is)
	updateTurn(&u)
	// send updates to all users
	err = updateClients(u)
	return err
}

func updateClients(u UpdateGroup) error {
	// When we enable view and ray tracing, we will only send what clients can see
	// we will always send any map tile updates, to all
	// we will check what upgraded players can see, and for the owner gets an update for all powerups in it's vision
	// we will see if clients can see a different owners upgraded player
	// Always send a YourTurn to who's turn it is, and "SoNSos_Turn" to everyone else
	// but for now, we will send all changes to all players

	// Always send a YourTurn to who's turn it is, and "SoNSos_Turn" to everyone else
	for i, c := range Clients {
		if i == ClientTurn {
			u.YourTurn = true
		} else {
			u.YourTurn = false
		}
		// Send the data
		json, err := u.MarshalJSON()
		if err != nil {
			return err
		}
		c.ConWrite <- json
	}
	return nil
}

func updateTurn(u *UpdateGroup) {
	changedTurn := false
	if len(Clients) >= 2 {
		if ClientTurn < 0 {
			ClientTurn = 0
			changedTurn = true
		} else if getClientMoves(ClientTurn) <= 0 {
			// The current client has used all their player's moves
			ClientTurn = (ClientTurn + 1) % int8(len(Clients))
			changedTurn = true
		}
	}
	if changedTurn {
		// Give the next client moves
		giveClientMoves(ClientTurn, u)
	}
	// Update the current_turn in u
	if len(Clients[ClientTurn].Nick) > 0 {
		u.ClientTurn = Clients[ClientTurn].Nick
	} else {
		u.ClientTurn = strconv.FormatInt(int64(ClientTurn), 10)
	}
}

func cleanString(ins string) string {
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		fmt.Printf("Could not compile regexp :%q\n", err)
		return ins
	}

	safe := reg.ReplaceAllString(ins, "-")
	safe = strings.ToLower(strings.Trim(safe, "-"))
	return safe
}

func GameLoop() {
	for {
		// Read from our clients
		for id, v := range Clients {
			select {
			case msg := <-v.ConRead:
				err := processCommand(id, msg)
				if err != nil {
					fmt.Printf("Got a bad command %s\n", msg)
					// Send error to the client who sent it
					//TODO
				}
			default:
				continue
			}
		}
	}
}

func StartGame() {
	// seed our random
	rand.Seed(time.Now().UnixNano())
	// Make our client map
	Clients = make(map[int8]GameClient)
	// Make our map of players
	GamePlayers = make([]Player, 0, 32)

	// run default commands from file
	// openfile
	f, err := os.Open(STARTUP_FILE)
	if err != nil {
		fmt.Printf("Could not open %q\n", STARTUP_FILE)
	}
	// run each command
	fread := bufio.NewReader(f)

	var cmdstr string
	for {
		cmdstr, err = fread.ReadString('\n')
		if len(cmdstr) == 0 {
			break
		}
		processCommand(-1, cmdstr[:len(cmdstr)-1])

		if err == io.EOF {
			break
		}
	}

	go GameLoop()
}
