package rush_n_crush

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

type Player struct {
	id           int8
	owner        int8
	pos          Position
	moves        int8
	direction    int16 // 0 is right, 180 or -180 are left
	defaultMoves int8
}

func (p Player) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"id\":")
	buf.WriteString(strconv.FormatInt(int64(p.id), 10))
	buf.WriteString(",\"owner\":")
	buf.WriteString(strconv.FormatInt(int64(p.owner), 10))
	buf.WriteString(",\"pos\":")
	posJSON, _ := p.pos.MarshalJSON()
	buf.Write(posJSON)
	buf.WriteString(",\"moves\":")
	buf.WriteString(strconv.FormatInt(int64(p.moves), 10))
	buf.WriteString(",\"dir\":")
	buf.WriteString(strconv.FormatInt(int64(p.direction), 10))
	buf.WriteString("}")
	return buf.Bytes(), nil
}

var GamePlayers []Player
var currentPlayerCount int8 = 0
var movesPerPlayer int8 = 6
var playersPerClient int8 = 2

func MovePlayer(arg_string string, id int8, u *UpdateGroup) error {
	var newx, newy int16
	var pid int8
	var t int64
	var err error

	argarr := strings.Split(arg_string, ",")

	t, err = strconv.ParseInt(argarr[0], 10, 8)
	if err != nil {
		return err
	}
	pid = int8(t)

	t, err = strconv.ParseInt(argarr[1], 10, 16)
	if err != nil {
		return err
	}
	newx = int16(t)

	t, err = strconv.ParseInt(argarr[2], 10, 16)
	if err != nil {
		return err
	}
	newy = int16(t)

	// check if the address is valid for moving
	if newx >= int16(len(GameMap[0])) || newx < 0 || newy >= int16(len(GameMap)) || newy < 0 {
		return errors.New("Out of Bounds")
	}
	if GameMap[newy][newx].tType != T_WALK || GameMap[newy][newx].occupied {
		return errors.New("Unmovable Tile")
	}

	var p int
	var found bool
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].id == pid {
			if GamePlayers[i].owner != id {
				return errors.New("Tried to move another user's player")
			}
			p = i
			found = true
		}
	}
	if found == false {
		return errors.New("Unknown Player")
	}

	if GamePlayers[p].moves <= 0 {
		return errors.New("Player out of Moves")
	}

	// check if the player tried to move by more than one space
	xdist := GamePlayers[p].pos.x - newx
	if xdist < 0 {
		xdist = 0 - xdist
	}
	ydist := GamePlayers[p].pos.y - newy
	if ydist < 0 {
		ydist = 0 - ydist
	}
	if ydist+xdist > 1 {
		return errors.New("Tried to move by more than one")
	}

	// Change occupied spot
	GameMap[GamePlayers[p].pos.y][GamePlayers[p].pos.x].occupied = false
	GameMap[newy][newx].occupied = true

	// Move
	GamePlayers[p].pos.x = newx
	GamePlayers[p].pos.y = newy

	// Take away a mov
	GamePlayers[p].moves -= 1

	// Add for updates
	u.PlayerUpdates = append(u.PlayerUpdates, GamePlayers[p])
	return nil
}

func AddPlayers(client int8, u *UpdateGroup) {
	players := make([]Player, playersPerClient)
	for i := 0; i < int(playersPerClient); i++ {
		var p Player
		p.id = currentPlayerCount
		currentPlayerCount++
		p.owner = client
		// TODO: Spawn locations
		p.pos = getRandomPosition()
		p.moves = 0
		p.defaultMoves = movesPerPlayer
		players[i] = p
		GameMap[p.pos.y][p.pos.x].occupied = true
	}
	GamePlayers = append(GamePlayers, players...)
	u.PlayerUpdates = append(u.PlayerUpdates, players...)
}

func getNumberPlayers(id int8) int8 {
	var num int8
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].owner == id {
			num++
		}
	}
	return num
}

func getClientMoves(id int8) int {
	var moves int
	for _, p := range GamePlayers {
		if p.owner == id {
			moves += int(p.moves)
		}
	}
	return moves
}

func clearClientMoves(id int8, u *UpdateGroup) {
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].owner == id {
			GamePlayers[i].moves = 0
			u.PlayerUpdates = append(u.PlayerUpdates, GamePlayers[i])
		}
	}
}

func giveClientMoves(id int8, u *UpdateGroup) {
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].owner == id {
			GamePlayers[i].moves = GamePlayers[i].defaultMoves
			u.PlayerUpdates = append(u.PlayerUpdates, GamePlayers[i])
		}
	}
}
