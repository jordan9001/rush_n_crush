package rush_n_crush

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	T_EMPTY int8 = 1
	T_SWALL int8 = 2
	T_WWALL int8 = 3
	T_SLOWV int8 = 4
	T_SLOWH int8 = 5
	T_WLOWV int8 = 6
	T_WLOWH int8 = 7
	T_WALK  int8 = 8
)

const (
	STARTUP_FILE string = "./run/startup.cmd"
)

type GameClient struct {
	Id       int8
	ConWrite chan string
	ConRead  chan string
}

type Position struct {
	x uint16
	y uint16
}

type Player struct {
	id      int8
	owner   int8
	pos     Position
	moves   int8
	updated int8
}

type Tile struct {
	pos      Position
	tType    int8
	health   int16
	nextType int8
	updated  int8
}

var Clients map[int8]GameClient
var GameMap [][]Tile
var update int8 = 0

// The syntax is command:json_object
func processCommand(id int8, message string) error {
	prev_update := update
	update = update + 1

	i := strings.Index(message, ":")
	if i < 0 {
		return errors.New("Got a bad command")
	}

	command := message[:i]
	switch command {
	case "get_gamestate":
		// They need to be sent the map

		// After sending, add in players for them
	case "map":
		// load a map
		LoadMap(message[i+1:])
	}
	// check if we should update state (who's turn it is)
	// send updates to all users
	updateClients(prev_update)

	return nil
}

func updateClients(prev_update int8) {
	// updated items have prev_update+1 as their updated field
	// if a player is upgraded, and it is the client's player check it against all objects, to see what it can see, and send those
	// if it isn't the clients, check if the client can see it, and send it if they can
	// Always send any map tile updates, regardless of position
	// Send the update command
}

func LoadMap(map_args string) error {
	maparr := strings.Split(map_args, ",")
	var w, h uint16
	var t int64
	var err error
	fmt.Printf("%s %s\n", maparr[0], maparr[1])

	t, err = strconv.ParseInt(maparr[0], 10, 16)
	if err != nil {
		fmt.Printf("Got err %q\n", err)
		return err
	}
	w = uint16(t)

	t, err = strconv.ParseInt(maparr[1], 10, 16)
	if err != nil {
		fmt.Printf("Got err %q\n", err)
		return err
	}
	h = uint16(t)

	fmt.Printf("Loading map of size %dx%d\n", w, h)

	// Allocate the map
	GameMap = make([][]Tile, h)
	for i := uint16(0); i < h; i++ {
		row := make([]Tile, w)
		for j := uint16(0); j < w; j++ {
			t, err = strconv.ParseInt(maparr[(i*w)+j+2], 10, 8)
			if err != nil {
				fmt.Printf("Got err %q\n", err)
				return err
			}
			var tile Tile
			tile.pos = Position{j, i}
			tile.tType = int8(t)
			row[j] = tile
			fmt.Printf("%d", t)
		}
		fmt.Printf("\n")
		GameMap[i] = row
	}
	return nil
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
	// Make our client map
	Clients = make(map[int8]GameClient)

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
