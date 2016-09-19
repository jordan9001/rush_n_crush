package rush_n_crush

import (
	"bytes"
	"strconv"
)

type Player struct {
	id           int8
	owner        int8
	pos          Position
	moves        int8
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
	buf.WriteString("}")
	return buf.Bytes(), nil
}

var GamePlayers []Player
var currentPlayerCount int8 = 0
var movesPerPlayer int8 = 6

func AddPlayers(client int8, u *UpdateGroup) {
	players := make([]Player, settings_playersPerClient)
	for i := 0; i < int(settings_playersPerClient); i++ {
		var p Player
		p.id = currentPlayerCount
		currentPlayerCount++
		p.owner = client
		// TODO: Spawn locations
		p.pos = getRandomPosition()
		p.moves = 0
		p.defaultMoves = movesPerPlayer
		players[i] = p
	}
	GamePlayers = append(GamePlayers, players...)
	u.PlayerUpdates = append(u.PlayerUpdates, players...)
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
	for _, p := range GamePlayers {
		if p.owner == id {
			p.moves = 0
			u.PlayerUpdates = append(u.PlayerUpdates, p)
		}
	}
}

func giveClientMoves(id int8, u *UpdateGroup) {
	for _, p := range GamePlayers {
		if p.owner == id {
			p.moves = p.defaultMoves
			u.PlayerUpdates = append(u.PlayerUpdates, p)
		}
	}
}
