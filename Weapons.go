package rush_n_crush

import (
	"bytes"
	"strconv"
)

type Weapon struct {
	name             string
	damage           func(start_x, start_y, direction, pdMult, tdMult int16, u *UpdateGroup) bool
	playerDamageMult int16
	tileDamageMult   int16
	damageType       string
	ammo             int16
	movesCost        int16
	pos              Position
}

func (w Weapon) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"name\":\"")
	buf.WriteString(w.name)
	buf.WriteString("\",\"ammo\":")
	buf.WriteString(strconv.FormatInt(int64(w.ammo), 10))
	buf.WriteString(",\"move_cost\":")
	buf.WriteString(strconv.FormatInt(int64(w.movesCost), 10))
	buf.WriteString(",\"pos\":")
	pos, _ := w.pos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

type HitInfo struct {
	damageType string
	pos        Position
}

var GameWeapons []Weapon

// damage functions
// should only be called if the weapon has enough ammo, and the player has enough moves
func damageStraight(start_x, start_y, direction, pdMult, tdMult int16, u *UpdateGroup) bool {
	// ray trace till we hit something
	hx, hy := traceDir(start_x, start_y, direction, true)
	var baseDamage int16 = 1
	// if it was occupied, hit the player
	// else hit the tile
	if GameMap[hy][hx].occupied == true {
		damagePlayer(hx, hy, baseDamage*pdMult, u)
	} else {
		damageTile(hx, hy, baseDamage*tdMult, u)
	}
	return true
}

func getPowerUp(x, y int16) (index int) {
	// return a weapon, if there is one
	found := false
	for index = 0; index < len(GameWeapons); index++ {
		if GameWeapons[index].pos.x == x && GameWeapons[index].pos.y == y {
			found = true
		}
	}
	if !found {
		index = -1
	}
	return index
}
