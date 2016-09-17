package rush_n_crush

import (
	"bytes"
	"strconv"
)

type Player struct {
	id    int8
	owner int8
	pos   Position
	moves int8
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
		players[i] = p
	}
	GamePlayers = append(GamePlayers, players...)
	u.PlayerUpdates = append(u.PlayerUpdates, players...)
}
