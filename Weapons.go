package rush_n_crush

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
)

type Weapon struct {
	name             string
	damage           func(start_x, start_y, direction int16, w Weapon, u *UpdateGroup) bool
	playerDamageMult int16
	tileDamageMult   int16
	damageType       string
	ammo             int16
	movesCost        int8
}

func (w Weapon) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"name\":\"")
	buf.WriteString(w.name)
	buf.WriteString("\",\"ammo\":")
	buf.WriteString(strconv.FormatInt(int64(w.ammo), 10))
	buf.WriteString(",\"move_cost\":")
	buf.WriteString(strconv.FormatInt(int64(w.movesCost), 10))
	buf.WriteString("}")
	return buf.Bytes(), nil
}

type WeaponCache []Weapon

func (wc WeaponCache) add(w Weapon) WeaponCache {
	// check first if there is one already
	for i := 0; i < len(wc); i++ {
		if wc[i].name == w.name {
			wc[i].ammo += w.ammo
			return wc
		}
	}
	return append(wc, w)
}

func (wc WeaponCache) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("[")
	first := true
	for _, w := range wc {
		if !first {
			buf.WriteString(",")
		}
		wep, _ := w.MarshalJSON()
		buf.Write(wep)
		first = false
	}
	buf.WriteString("]")
	return buf.Bytes(), nil
}

type HitInfo struct {
	damageType string
	pos        Position
}

func (h HitInfo) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"damage_type\":\"")
	buf.WriteString(h.damageType)
	buf.WriteString("\",\"pos\":")
	pos, _ := h.pos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

var GameWeapons []Weapon

// damage functions
// should only be called if the weapon has enough ammo, and the player has enough moves
func damageStraight(start_x, start_y, direction int16, w Weapon, u *UpdateGroup) bool {
	// Add some random to the shots
	rand_max := 24
	direction = direction + int16(rand.Intn(rand_max)-(rand_max/2))
	// ray trace till we hit something
	hx, hy := traceDir(start_x, start_y, direction, true)
	fmt.Printf("Shot %d,%d\n", hx, hy)
	// Update for animation
	u.WeaponHits = append(u.WeaponHits, HitInfo{damageType: w.damageType, pos: Position{x: hx, y: hy}})
	var baseDamage int16 = 1
	// else hit the tile
	if GameMap[hy][hx].occupied == true {
		damagePlayer(hx, hy, baseDamage*w.playerDamageMult, u)
	} else {
		damageTile(hx, hy, baseDamage*w.tileDamageMult, u)
	}
	return true
}
