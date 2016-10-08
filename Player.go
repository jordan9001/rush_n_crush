package rush_n_crush

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Player struct {
	id           int8
	owner        int
	pos          Position
	health       int16
	maxHealth    int16
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
	buf.WriteString(",\"max_health\":")
	buf.WriteString(strconv.FormatInt(int64(p.maxHealth), 10))
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

func AddPlayers(client int, gv *GameVariables) {
	for i := 0; i < int(gv.playersPerClient); i++ {
		var p Player
		p.id = gv.currentPlayerCount
		gv.currentPlayerCount++
		p.owner = client
		// TODO: Spawn locations
		p.pos = getRandomPosition(gv)
		p.moves = 0
		p.health = gv.defaultPlayerHealth
		p.maxHealth = p.health
		p.defaultMoves = gv.movesPerPlayer
		// add default weapon
		p.weapons = make([]Weapon, 0, 3)
		w := pistol.makeCopy()
		p.weapons = p.weapons.add(w)
		w = shovel.makeCopy()
		p.weapons = p.weapons.add(w)
		w = bazooka.makeCopy()
		p.weapons = p.weapons.add(w)
		gv.GamePlayers = append(gv.GamePlayers, p)
		gv.GameMap[p.pos.y][p.pos.x].occupied = true
	}
}

func MovePlayer(arg_string string, id int, u *UpdateGroup, gv *GameVariables) error {
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
	if newx >= int16(len(gv.GameMap[0])) || newx < 0 || newy >= int16(len(gv.GameMap)) || newy < 0 {
		return errors.New("Out of Bounds")
	}
	if gv.GameMap[newy][newx].tType != T_WALK || gv.GameMap[newy][newx].occupied {
		return errors.New("Unmovable Tile")
	}

	var p int
	var found bool
	for i := 0; i < len(gv.GamePlayers); i++ {
		if gv.GamePlayers[i].id == pid {
			if gv.GamePlayers[i].owner != id {
				return errors.New("Tried to move another user's player")
			}
			p = i
			found = true
		}
	}
	if found == false {
		return errors.New("Unknown Player")
	}

	if gv.GamePlayers[p].moves <= 0 {
		return errors.New("Player out of Moves")
	}

	// check if the player tried to move by more than one space
	xdist := gv.GamePlayers[p].pos.x - newx
	if xdist < 0 {
		xdist = 0 - xdist
	}
	ydist := gv.GamePlayers[p].pos.y - newy
	if ydist < 0 {
		ydist = 0 - ydist
	}
	if ydist+xdist != 1 {
		return errors.New("Didn't move by one")
	}

	// Change occupied spot
	gv.GameMap[gv.GamePlayers[p].pos.y][gv.GamePlayers[p].pos.x].occupied = false
	gv.GameMap[newy][newx].occupied = true

	// Move
	gv.GamePlayers[p].pos.x = newx
	gv.GamePlayers[p].pos.y = newy
	gv.GamePlayers[p].direction = dir

	// Take away a mov
	gv.GamePlayers[p].moves -= 1

	// Check if there are powerups
	got_powerups := false
	for i := 0; i < len(gv.GameMap[newy][newx].powerups); i++ {
		gv.GamePlayers[p].weapons = gv.GamePlayers[p].weapons.add(gv.GameMap[newy][newx].powerups[i])
		got_powerups = true
	}
	if got_powerups {
		// remove the cache
		gv.GameMap[newy][newx].powerups = nil
		// update the tile
		u.TileUpdates = append(u.TileUpdates, gv.GameMap[newy][newx])
	}
	return nil
}

func fire(message string, client int, u *UpdateGroup, gv *GameVariables) error {
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
	for p := 0; p < len(gv.GamePlayers); p++ {
		if pid == gv.GamePlayers[p].id {
			if gv.GamePlayers[p].owner != client {
				return errors.New("Someone tried to move a player that is not theirs")
			}
			// move the players dir
			gv.GamePlayers[p].direction = dir
			// find the weapon
			for w := 0; w < len(gv.GamePlayers[p].weapons); w++ {
				// check the, moves
				if gv.GamePlayers[p].weapons[w].name == weaponArg {
					if gv.GamePlayers[p].moves < gv.GamePlayers[p].weapons[w].movesCost {
						return errors.New("Player doesn't have enough moves to fire this weapon")
					} else if gv.GamePlayers[p].weapons[w].ammo == 0 {
						return errors.New("Out of Ammo")
					}
					// fire it
					if gv.GamePlayers[p].weapons[w].damage(gv.GamePlayers[p].pos.x, gv.GamePlayers[p].pos.y, dir, gv.GamePlayers[p].weapons[w], u, gv) {
						fmt.Printf("Shot something!\n")
						// a Player may have died, we need to get p again
						foundp := false
						for newp := 0; newp < len(gv.GamePlayers); newp++ {
							if pid == gv.GamePlayers[newp].id {
								p = newp
								foundp = true
							}
						}
						if !foundp {
							// dude killed itself
							return nil
						}
						if gv.GamePlayers[p].weapons[w].ammo > 0 {
							gv.GamePlayers[p].weapons[w].ammo -= 1
						}
						gv.GamePlayers[p].moves -= gv.GamePlayers[p].weapons[w].movesCost
					}
					return nil
				}
			}
		}
	}
	return errors.New("Couldn't find the player/weapon combo")
}

func damagePlayer(x, y, damage int16, u *UpdateGroup, gv *GameVariables) {
	// find the player
	for i := 0; i < len(gv.GamePlayers); i++ {
		if gv.GamePlayers[i].pos.x == x && gv.GamePlayers[i].pos.y == y {
			gv.GamePlayers[i].health -= damage
			if gv.GamePlayers[i].health <= 0 {
				// remove player
				gv.GameMap[gv.GamePlayers[i].pos.y][gv.GamePlayers[i].pos.x].occupied = false
				gv.GamePlayers = append(gv.GamePlayers[:i], gv.GamePlayers[i+1:]...)

			}
		}
	}
}

func makePlayerUpdates(client int, gv *GameVariables) map[int8]Player {
	playerUpdates := make(map[int8]Player)
	// for each of this clients players
	for cp := 0; cp < len(gv.GamePlayers); cp++ {
		if gv.GamePlayers[cp].owner == client {
			// add it to the map
			playerUpdates[gv.GamePlayers[cp].id] = gv.GamePlayers[cp]
			x := gv.GamePlayers[cp].pos.x
			y := gv.GamePlayers[cp].pos.y
			// add everything it can see
			for o := 0; o < len(gv.GamePlayers); o++ {
				if gv.GamePlayers[o].owner != client {
					sx, sy := trace(x, y, gv.GamePlayers[o].pos.x, gv.GamePlayers[o].pos.y, false, gv)
					if sx == gv.GamePlayers[o].pos.x && sy == gv.GamePlayers[o].pos.y {
						playerUpdates[gv.GamePlayers[o].id] = gv.GamePlayers[o]
					}
				}
			}
		}
	}
	return playerUpdates
}

func getNumberPlayers(id int, gv *GameVariables) int8 {
	var num int8
	for i := 0; i < len(gv.GamePlayers); i++ {
		if gv.GamePlayers[i].owner == id {
			num++
		}
	}
	return num
}

func getClientMoves(id int, gv *GameVariables) int {
	var moves int
	for _, p := range gv.GamePlayers {
		if p.owner == id {
			moves += int(p.moves)
		}
	}
	return moves
}

func clearClientMoves(id int, gv *GameVariables) {
	for i := 0; i < len(gv.GamePlayers); i++ {
		if gv.GamePlayers[i].owner == id {
			gv.GamePlayers[i].moves = 0
		}
	}
}

func giveClientMoves(id int, gv *GameVariables) {
	for i := 0; i < len(gv.GamePlayers); i++ {
		if gv.GamePlayers[i].owner == id {
			if gv.turnNumber <= 1 {
				gv.GamePlayers[i].moves = gv.GamePlayers[i].defaultMoves / 2
			} else {
				gv.GamePlayers[i].moves = gv.GamePlayers[i].defaultMoves
			}
		}
	}
}
