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
	health       int16
	moves        int8
	direction    int16 // 0 is right, 180 or -180 are left, 90 is down
	defaultMoves int8
	weapons      WeaponCache
}

func (p Player) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"id\":")
	buf.WriteString(strconv.FormatInt(int64(p.id), 10))
	buf.WriteString(",\"owner\":")
	buf.WriteString(strconv.FormatInt(int64(p.owner), 10))
	buf.WriteString(",\"pos\":")
	posJSON, _ := p.pos.MarshalJSON()
	buf.Write(posJSON)
	buf.WriteString(",\"health\":")
	buf.WriteString(strconv.FormatInt(int64(p.health), 10))
	buf.WriteString(",\"moves\":")
	buf.WriteString(strconv.FormatInt(int64(p.moves), 10))
	buf.WriteString(",\"dir\":")
	buf.WriteString(strconv.FormatInt(int64(p.direction), 10))
	buf.WriteString(",\"weapons\":[")
	first := true
	for _, v := range p.weapons {
		if !first {
			buf.WriteString(",")
		}
		weaponJSON, _ := v.MarshalJSON()
		buf.Write(weaponJSON)
		first = false
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

var GamePlayers []Player
var currentPlayerCount int8 = 0
var movesPerPlayer int8 = 12
var defaultPlayerHealth int16 = 100
var playersPerClient int8 = 3

func MovePlayer(arg_string string, id int8, u *UpdateGroup) error {
	var newx, newy, dir int16
	var pid int8
	var t int64
	var err error

	argarr := strings.Split(arg_string, ",")
	if len(argarr) < 4 {
		return errors.New("Not enough arguments to player_move")
	}

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

	t, err = strconv.ParseInt(argarr[3], 10, 16)
	if err != nil {
		return err
	}
	dir = int16(t)

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
	if ydist+xdist != 1 {
		return errors.New("Didn't move by one")
	}

	// Change occupied spot
	GameMap[GamePlayers[p].pos.y][GamePlayers[p].pos.x].occupied = false
	GameMap[newy][newx].occupied = true

	// Move
	GamePlayers[p].pos.x = newx
	GamePlayers[p].pos.y = newy
	GamePlayers[p].direction = dir

	// Take away a mov
	GamePlayers[p].moves -= 1

	// Check if there are powerups
	got_powerups := false
	for i := 0; i < len(GameMap[newy][newx].powerups); i++ {
		GamePlayers[p].weapons = GamePlayers[p].weapons.add(GameMap[newy][newx].powerups[i])
		got_powerups = true
	}
	if got_powerups {
		// remove the cache
		GameMap[newy][newx].powerups = nil
		// update the tile
		u.TileUpdates = append(u.TileUpdates, GameMap[newy][newx])
	}
	return nil
}

func fire(message string, client int8, u *UpdateGroup) error {
	var pid int8
	var weaponArg string
	var dir int16

	argarr := strings.Split(message, ",")
	if len(argarr) < 3 {
		return errors.New("Not enough arguments to fire")
	}

	t, err := strconv.ParseInt(argarr[0], 10, 8)
	if err != nil {
		return err
	}
	pid = int8(t)

	weaponArg = argarr[1]

	t, err = strconv.ParseInt(argarr[2], 10, 16)
	if err != nil {
		return err
	}
	dir = int16(t)

	// find the player
	for p := 0; p < len(GamePlayers); p++ {
		if pid == GamePlayers[p].id {
			if GamePlayers[p].owner != client {
				return errors.New("Someone tried to move a player that is not theirs")
			}
			// move the players dir
			GamePlayers[p].direction = dir
			// find the weapon
			for w := 0; w < len(GamePlayers[p].weapons); w++ {
				// check the, moves
				if GamePlayers[p].weapons[w].name == weaponArg {
					if GamePlayers[p].moves < GamePlayers[p].weapons[w].movesCost {
						return errors.New("Player doesn't have enough moves to fire this weapon")
					} else if GamePlayers[p].weapons[w].ammo == 0 {
						return errors.New("Out of Ammo")
					}
					// fire it
					GamePlayers[p].weapons[w].damage(GamePlayers[p].pos.x, GamePlayers[p].pos.y, dir, GamePlayers[p].weapons[w], u)
					if GamePlayers[p].weapons[w].ammo > 0 {
						GamePlayers[p].weapons[w].ammo -= 1
					}
					GamePlayers[p].moves -= GamePlayers[p].weapons[w].movesCost
					return nil
				}
			}
		}
	}
	return errors.New("Couldn't find the player/weapon combo")
}

func damagePlayer(x, y, damage int16, u *UpdateGroup) {
	// find the player
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].pos.x == x && GamePlayers[i].pos.y == y {
			GamePlayers[i].health -= damage
			if GamePlayers[i].health <= 0 {
				// remove player
				GameMap[GamePlayers[i].pos.y][GamePlayers[i].pos.x].occupied = false
				GamePlayers = append(GamePlayers[:i], GamePlayers[i+1:]...)

			}
		}
	}
}

func AddPlayers(client int8) {
	for i := 0; i < int(playersPerClient); i++ {
		var p Player
		p.id = currentPlayerCount
		currentPlayerCount++
		p.owner = client
		// TODO: Spawn locations
		p.pos = getRandomPosition()
		p.moves = 0
		p.health = defaultPlayerHealth
		p.defaultMoves = movesPerPlayer
		// add default weapon
		p.weapons = make([]Weapon, 0, 3)
		w := Weapon{
			name:             "pistol",
			damage:           damageStraight,
			playerDamageMult: 25,
			tileDamageMult:   30,
			damageType:       "bullet",
			ammo:             -1,
			movesCost:        4,
		}
		p.weapons = p.weapons.add(w)
		GamePlayers = append(GamePlayers, p)
		GameMap[p.pos.y][p.pos.x].occupied = true
	}
}

func makePlayerUpdates(client int8) map[int8]Player {
	playerUpdates := make(map[int8]Player)
	// for each of this clients players
	for cp := 0; cp < len(GamePlayers); cp++ {
		if GamePlayers[cp].owner == client {
			// add it to the map
			playerUpdates[GamePlayers[cp].id] = GamePlayers[cp]
			x := GamePlayers[cp].pos.x
			y := GamePlayers[cp].pos.y
			// add everything it can see
			for o := 0; o < len(GamePlayers); o++ {
				if GamePlayers[o].owner != client {
					sx, sy := trace(x, y, GamePlayers[o].pos.x, GamePlayers[o].pos.y, false)
					if sx == GamePlayers[o].pos.x && sy == GamePlayers[o].pos.y {
						playerUpdates[GamePlayers[o].id] = GamePlayers[o]
					}
				}
			}
		}
	}
	return playerUpdates
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

func clearClientMoves(id int8) {
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].owner == id {
			GamePlayers[i].moves = 0
		}
	}
}

func giveClientMoves(id int8) {
	for i := 0; i < len(GamePlayers); i++ {
		if GamePlayers[i].owner == id {
			if turnNumber <= 1 {
				GamePlayers[i].moves = GamePlayers[i].defaultMoves / 2
			} else {
				GamePlayers[i].moves = GamePlayers[i].defaultMoves
			}
		}
	}
}
