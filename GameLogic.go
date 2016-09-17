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
	ConWrite chan string
	ConRead  chan string
	Nick     string
}

type UpdateGroup struct {
	YourTurn      bool
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
	buf.WriteString(",\"updated_tiles\":[")
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
	case "map":
		// if a game has not been started, load a map
		if ClientTurn < 0 {
			LoadMap(message[i+1:])
		}
	case "setting_players_per_client":
		if id == -1 {
			desired, _ := strconv.ParseInt(message[i+1:], 10, 8)
			if desired > 0 {
				settings_playersPerClient = int8(desired)
			}
		}
	case "end_turn":
		// Move to next Client
	}
	// check if we should update state (who's turn it is)
	if len(Clients) >= 2 {
		if ClientTurn < 0 {
			ClientTurn = 0
		}
		// Check if the current client has used all their moves
	}
	// send updates to all users
	updateClients(u)

	return nil
}

func updateClients(u UpdateGroup) {
	// When we enable view and ray tracing, we will only send what clients can see
	// we will always send any map tile updates, to all
	// we will check what upgraded players can see, and for the owner gets an update for all powerups in it's vision
	// we will see if clients can see a different owners upgraded player
	// Always send a YourTurn to who's turn it is, and "SoNSos_Turn" to everyone else
	// but for now, we will send all changes to all players
	// Always send a YourTurn to who's turn it is, and "SoNSos_Turn" to everyone else

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
