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

var GameCounter int = 0
var Clients map[int]GameClient

type GameClient struct {
	Id         int
	ConWrite   chan []byte
	Nick       string
	GameNumber int
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
	YourId        int
	ClientTurn    int
	TileUpdates   []Tile
	PlayerUpdates map[int8]Player
	PowerUpdates  map[int32]PowerUp // mapped by ((pos.x << 16) & (pos.y))
	WeaponHits    []HitInfo
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
	buf.WriteString("],\"powerups\":[")
	first = true
	for _, v := range u.PowerUpdates {
		if !first {
			buf.WriteString(",")
		}
		puJSON, _ := v.MarshalJSON()
		buf.Write(puJSON)
		first = false
	}
	buf.WriteString("],\"hit_tiles\":[")
	first = true
	for _, v := range u.WeaponHits {
		if !first {
			buf.WriteString(",")
		}
		HitJSON, _ := v.MarshalJSON()
		buf.Write(HitJSON)
		first = false
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

type GameVariables struct {
	GameNumber          int
	GameMap             [][]Tile
	Spawns              []Spawn
	GamePlayers         []Player
	currentPlayerCount  int8
	movesPerPlayer      int8
	defaultPlayerHealth int16
	playersPerClient    int8
	ClientsForGame      int
	ClientsInGame       int
	ClientTurn          int
	turnNumber          int
	PowerUps            []PowerUp
	puplevel0           WeaponCache
	pup0refresh         int
	puplevel1           WeaponCache
	pup1refresh         int
	puplevel2           WeaponCache
	pup2refresh         int
}

// The syntax is command:comma,separated,args
func processCommand(id int, message string, gv *GameVariables) error {
	var err error = nil
	// preallocate the update group
	var u UpdateGroup
	u.TileUpdates = make([]Tile, 0, 16)
	u.WeaponHits = make([]HitInfo, 0, 1)

	i := strings.Index(message, ":")
	if i < 0 {
		return errors.New("Got a bad command")
	}

	command := message[:i]
	fmt.Printf("Got command %q\n", command)

	// Run Commands
	switch command {
	case "get_gamestate": // no args
		// Register client, but just wait in lobby until the admin sends the start command
		// From the lobby they can change map, change number of clients, change game type, etc
		// TODO
		// Till then, just add to the game
		// TODO make sure they only call this once
		fmt.Printf("Sending gamestate to client %d\n", id)
		gv.ClientsInGame++
		// see if we have a spawn for them
		perm := rand.Perm(len(gv.Spawns))
		for i, _ := range perm {
			if gv.Spawns[i].client < 0 {
				gv.Spawns[i].client = id
				break
			}
		}
		// They need to be sent the map
		SendMap(id, gv)
		// After sending, add in players for them
		AddPlayers(id, gv)
	case "player_move": // args = player_id, newx, newy, dir
		// moves player, and updates dir
		err = MovePlayer(message[i+1:], id, &u, gv)
		if err != nil {
			return err
		}
	case "fire": // args = player_id, weapon, dir
		err = fire(message[i+1:], id, &u, gv)
		if err != nil {
			return err
		}
	case "set_nick": // arg = nick_string
		// Set this client's nickname
		c := Clients[id]
		c.Nick = cleanString(message[i+1:])
		Clients[id] = c
		// Send everyone a who's who
		sendWhosWho(-1, gv)
		return nil
	case "who_is_who":
		// Send the requester a who's who
		sendWhosWho(id, gv)
		return nil
	case "map": // args = width,height,tile_typex0y0,title_typex1y0,...
		// if a game has not been started, load a map
		if gv.ClientTurn < 0 {
			LoadMap(message[i+1:], gv)
		}
	case "set_default_moves": // arg = default numb
		if id == -1 {
			var desired int64
			desired, err = strconv.ParseInt(message[i+1:], 10, 8)
			if err != nil {
				return err
			}
			if desired > 0 {
				gv.movesPerPlayer = int8(desired)
			}
		}
		return nil
	case "set_players_per_client": // arg = player_per_client
		if id == -1 {
			var desired int64
			desired, err = strconv.ParseInt(message[i+1:], 10, 8)
			if err != nil {
				return err
			}
			if desired > 0 {
				gv.playersPerClient = int8(desired)
			}
		}
		return nil
	case "set_clients_per_game": // arg = player_per_client
		if id == -1 {
			var desired int64
			desired, err = strconv.ParseInt(message[i+1:], 10, 64)
			if err != nil {
				return err
			}
			if desired > 0 {
				gv.ClientsForGame = int(desired)
			}
		}
		return nil
	case "end_turn": // no arg
		// Move to next Client
		clearClientMoves(id, gv)
	case "DISCONNECTED":
		clearClientMoves(id, gv)
		// remove the players
		clearPlayers(id, gv)
		// remove the client
		gv.ClientsInGame--
		delete(Clients, id)
	}
	// check if we should update state (who's turn it is)
	updateTurn(gv)
	// send updates to all users
	err = updateClients(u, gv)
	return err
}

func updateClients(u UpdateGroup, gv *GameVariables) error {
	var client_u UpdateGroup
	for i, currentClient := range Clients {
		if currentClient.GameNumber != gv.GameNumber {
			continue
		}
		// if the player has had everyone die, still show them whats happening
		client_u = UpdateGroup{
			YourId:      currentClient.Id,
			ClientTurn:  gv.ClientTurn,
			TileUpdates: u.TileUpdates,
			WeaponHits:  u.WeaponHits,
		}
		// if the player has had everyone die, still show them whats happening
		if getNumberPlayers(i, gv) > 0 {
			client_u.PlayerUpdates, client_u.PowerUpdates = makePlayerUpdates(currentClient.Id, gv)
		} else if gv.turnNumber > 1 {
			client_u.PlayerUpdates, client_u.PowerUpdates = makePlayerUpdates(-1, gv)
		}
		// Send the data
		data, _ := client_u.MarshalJSON()
		//fmt.Printf("%d sees %q\n\n", currentClient.Id, data)
		fmt.Printf("Sent update to %d\n", currentClient.Id)
		m := Message{"update", data}
		json, _ := m.MarshalJSON()
		currentClient.ConWrite <- json
	}
	return nil
}

func updateTurn(gv *GameVariables) {
	changedTurn := false
	if gv.ClientsInGame >= gv.ClientsForGame && len(gv.GamePlayers) > 0 {
		if gv.ClientTurn < 0 {
			gv.ClientTurn = 0
			changedTurn = true
		} else if getClientMoves(gv.ClientTurn, gv) <= 0 {
			next := false
			for {
				for i, v := range Clients {
					if next {
						if gv.ClientTurn == i || (gv.GameNumber == v.GameNumber && getNumberPlayers(i, gv) > 0) {
							gv.ClientTurn = i
							next = false
							break
						}
					} else if i == gv.ClientTurn {
						next = true
					}
				}
				if next == false {
					break
				}
			}
			changedTurn = true
		}
	}
	if changedTurn {
		gv.turnNumber = gv.turnNumber + 1
		fmt.Printf("\tTurn : %d\n", gv.turnNumber)
		// Give the next client moves
		giveClientMoves(gv.ClientTurn, gv)
		// add powerups
		updatePowerups(gv)
	}
}

func sendWhosWho(id int, gv *GameVariables) {
	data := bytes.NewBufferString("{")
	first := true
	for _, v := range Clients {
		if v.GameNumber != gv.GameNumber {
			continue
		}
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
		if v.GameNumber != gv.GameNumber {
			continue
		}
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

func GameLoop(gv *GameVariables) {
	for {
		// Read from our clients
		msg := <-con_read[gv.GameNumber]
		err := processCommand(msg.client, msg.message, gv)
		if err != nil {
			fmt.Printf("Got error \"%v\" for command %s\n", err, msg.message)
			// Send error to the client who sent it
			//TODO
		}
	}
}

func StartGame(startup_path string) (int, error) {
	// seed our random
	rand.Seed(time.Now().UnixNano())
	// Make the game variables
	var gv GameVariables
	gv.GameNumber = GameCounter
	GameCounter++
	// Make our client map
	if len(Clients) == 0 {
		Clients = make(map[int]GameClient)
	}
	// make our powerups
	gv.PowerUps = make([]PowerUp, 0, 8)
	// Make defaults
	gv.currentPlayerCount = 0
	gv.movesPerPlayer = 12
	gv.defaultPlayerHealth = 100
	gv.playersPerClient = 3
	gv.ClientsForGame = 2
	// set initial
	gv.ClientsInGame = 0
	gv.ClientTurn = -1
	gv.turnNumber = 0

	var shotgunCache WeaponCache = make([]Weapon, 0, 2)
	shotgunCache = shotgunCache.add(shotgun)
	shotgunCache = shotgunCache.add(sniper)
	var rocketCache WeaponCache = make([]Weapon, 0, 2)
	rocketCache = rocketCache.add(bazooka)
	rocketCache = rocketCache.add(bazooka)
	rocketCache = rocketCache.add(suicide)
	var wallCache WeaponCache = make([]Weapon, 0, 2)
	wallCache = wallCache.add(eztrump)
	wallCache = wallCache.add(minecraft)
	gv.puplevel0 = shotgunCache
	gv.pup0refresh = -1
	gv.puplevel1 = rocketCache
	gv.pup1refresh = -1
	gv.puplevel2 = wallCache
	gv.pup2refresh = -1

	// Make our read chan
	c := make(chan command)
	con_read[gv.GameNumber] = c
	// Make our map of players
	gv.GamePlayers = make([]Player, 0, int(gv.playersPerClient)*gv.ClientsForGame)
	gv.Spawns = make([]Spawn, 0, gv.ClientsForGame)

	// run default commands from file
	if len(startup_path) > 0 {
		// openfile
		f, err := os.Open(startup_path)
		if err != nil {
			fmt.Printf("Could not open %q\n", startup_path)
			return -1, errors.New("Bad startup command file")
		}
		// run each command
		fread := bufio.NewReader(f)

		var cmdstr string
		for {
			cmdstr, err = fread.ReadString('\n')
			if len(cmdstr) == 0 {
				break
			}
			processCommand(-1, cmdstr[:len(cmdstr)-1], &gv)

			if err == io.EOF {
				break
			}
		}
	}

	go GameLoop(&gv)
	return gv.GameNumber, nil
}
