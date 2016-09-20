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

type GameClient struct {
	Id       int8
	ConWrite chan []byte
	Nick     string
}

type Message struct {
	Type string
	Data []byte
}

func (m Message) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"message_type\":\"")
	buf.WriteString(m.Type)
	buf.WriteString("\",\"data\":")
	buf.Write(m.Data)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

type UpdateGroup struct {
	YourId        int8
	ClientTurn    int8
	TileUpdates   []Tile
	PlayerUpdates []Player
}

func (u UpdateGroup) MarshalJSON() ([]byte, error) {
	var first bool
	buf := bytes.NewBufferString("{\"your_id\":")
	buf.WriteString(strconv.FormatInt(int64(u.YourId), 10))
	buf.WriteString(",\"current_turn\":\"")
	buf.WriteString(strconv.FormatInt(int64(u.ClientTurn), 10))
	buf.WriteString("\",\"updated_tiles\":[")
	first = true
	for _, v := range u.TileUpdates {
		if !first {
			buf.WriteString(",")
		}
		tileJSON, _ := v.MarshalJSON()
		buf.Write(tileJSON)
		first = false
	}
	buf.WriteString("],\"updated_players\":[")
	first = true
	for _, v := range u.PlayerUpdates {
		if !first {
			buf.WriteString(",")
		}
		playerJSON, _ := v.MarshalJSON()
		buf.Write(playerJSON)
		first = false
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

// Game State Variables
var Clients map[int8]GameClient
var ClientTurn int8 = -1
var settings_playersPerClient int8 = 1

// The syntax is command:comma,separated,args
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
	case "get_gamestate": // no args
		// They need to be sent the map
		SendMap(id)
		// After sending, add in players for them
		AddPlayers(id, &u)
	case "player_move": // args = player_id, newx, newy
		err = MovePlayer(message[i+1:], id, &u)
		if err != nil {
			return err
		}
	case "player_dir": // arg = player_dir

	case "set_nick": // arg = nick_string
		// Set this client's nickname
		c := Clients[id]
		c.Nick = cleanString(message[i+1:])
		// Send everyone a who's who
		sendWhosWho(-1)
		return nil
	case "who_is_who":
		// Send the requester a who's who
		sendWhosWho(id)
		return nil
	case "set_default_moves": // arg = default numb
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
	case "map": // args = width,height,tile_type,...
		// if a game has not been started, load a map
		if ClientTurn < 0 {
			LoadMap(message[i+1:])
		}
	case "setting_players_per_client": // arg = player_per_client
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
	case "end_turn": // no arg
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
	// we will check what upgraded players can see, and the owner gets an update for all objects and players in it's vision
	// we will see if clients can see a different owners upgraded player
	// but for now, we will send all changes to all players
	for _, c := range Clients {
		u.YourId = c.Id
		// Send the data
		data, _ := u.MarshalJSON()
		m := Message{"update", data}
		json, _ := m.MarshalJSON()
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
	u.ClientTurn = ClientTurn
}

func sendWhosWho(id int8) {
	data := bytes.NewBufferString("{")
	first := true
	for _, v := range Clients {
		if !first {
			data.WriteString(",")
		}
		data.WriteRune('"')
		data.WriteString(strconv.FormatInt(int64(v.Id), 10))
		data.WriteString("\":\"")
		data.WriteString(v.Nick)
		first = false
	}

	// if id < 0, send it to everyone
	sendable := data.Bytes()
	for _, v := range Clients {
		if id < 0 || v.Id == id {
			v.ConWrite <- sendable
		}
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
		msg := <-con_read
		err := processCommand(msg.client, msg.message)
		if err != nil {
			fmt.Printf("Got error \"%v\" for command %s\n", err, msg.message)
			// Send error to the client who sent it
			//TODO
		}
	}
}

func StartGame(startup_path string) (chan command, error) {
	// seed our random
	rand.Seed(time.Now().UnixNano())
	// Make our client map
	Clients = make(map[int8]GameClient)
	// Make our read chan
	c := make(chan command)
	// Make our map of players
	GamePlayers = make([]Player, 0, 32)

	// run default commands from file
	if len(startup_path) > 0 {
		// openfile
		f, err := os.Open(startup_path)
		if err != nil {
			fmt.Printf("Could not open %q\n", startup_path)
			return nil, errors.New("Bad startup command file")
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
	}

	go GameLoop()
	return c, nil
}
