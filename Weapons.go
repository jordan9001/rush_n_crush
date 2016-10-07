package rush_n_crush

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
)

type Weapon struct {
	name             string
	damage           func(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool
	playerDamageMult int16
	tileDamageMult   int16
	damageType       string
	ammo             int16
	movesCost        int8
	distance         int16
}

func (w Weapon) makeCopy() (r Weapon) {
	r = Weapon{
		name:             w.name,
		damage:           w.damage,
		playerDamageMult: w.playerDamageMult,
		tileDamageMult:   w.tileDamageMult,
		damageType:       w.damageType,
		ammo:             w.ammo,
		movesCost:        w.movesCost,
		distance:         w.distance,
	}
	return
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
	fromPos    Position
}

func (h HitInfo) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"damage_type\":\"")
	buf.WriteString(h.damageType)
	buf.WriteString("\",\"pos\":")
	pos, _ := h.pos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString(",\"from_pos\":")
	pos, _ = h.fromPos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

var GameWeapons []Weapon

// damage functions
// should only be called if the weapon has enough ammo, and the player has enough moves
func damageStraight(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	// Add some random to the shots
	rand_max := 24
	direction = direction + int16(rand.Intn(rand_max)-(rand_max/2))
	// ray trace till we hit something
	hx, hy := traceDir(start_x, start_y, direction, w.distance, true, gv)
	fmt.Printf("Shot %d,%d\n", hx, hy)
	// Update for animation
	u.WeaponHits = append(u.WeaponHits, HitInfo{damageType: w.damageType, pos: Position{x: hx, y: hy}, fromPos: Position{x: start_x, y: start_y}})
	var baseDamage int16 = 1
	// else hit the tile
	if gv.GameMap[hy][hx].occupied == true {
		damagePlayer(hx, hy, baseDamage*w.playerDamageMult, u, gv)
	} else {
		damageTile(hx, hy, baseDamage*w.tileDamageMult, u, gv)
	}
	return true
}

// Weapons

var pistol Weapon = Weapon{
	name:             "pistol",
	damage:           damageStraight,
	playerDamageMult: 25,
	tileDamageMult:   30,
	damageType:       "bullet",
	ammo:             -1,
	movesCost:        4,
	distance:         64,
}