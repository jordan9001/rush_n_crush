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
	randomAngle      int
	shotsPerShot     int
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
		randomAngle:      w.randomAngle,
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
	buf.WriteString(",\"damageType\":\"")
	buf.WriteString(w.damageType)
	buf.WriteString("\"}")
	return buf.Bytes(), nil
}

type WeaponCache []Weapon

func (wc WeaponCache) add(w Weapon) WeaponCache {
	// check first if there is one already
	for i := 0; i < len(wc); i++ {
		if wc[i].name == w.name {
			if w.ammo == -1 || wc[i].ammo == -1 {
				wc[i].ammo = -1
			} else {
				wc[i].ammo += w.ammo
			}
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

type PowerUp struct {
	weapons         WeaponCache
	possibleWeapons WeaponCache
	pos             Position
	refresh         int
	lastRefresh     int
}

func (pu PowerUp) getId() int32 {
	var id int32 = (int32(pu.pos.x) << 16) | int32(pu.pos.y)
	return id
}

func (pu PowerUp) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"weapons\":")
	p, _ := pu.weapons.MarshalJSON()
	buf.Write(p)
	buf.WriteString(",\"pos\":")
	pos, _ := pu.pos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString("}")
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

func updatePowerups(gv *GameVariables) bool {
	// check each powerup
	for i := 0; i < len(gv.PowerUps); i++ {
		if gv.PowerUps[i].refresh == -1 {
			if gv.PowerUps[i].lastRefresh <= 0 {
				// One time powerups, add all in possible
				if len(gv.PowerUps[i].weapons) == 0 {
					gv.PowerUps[i].weapons = gv.PowerUps[i].possibleWeapons
					fmt.Printf("Updated PowerUp at %d,%d\n", gv.PowerUps[i].pos.x, gv.PowerUps[i].pos.y)
				}
				gv.PowerUps[i].lastRefresh = 1
			} else {
				// Remove the powerup
				gv.PowerUps = append(gv.PowerUps[:i], gv.PowerUps[i+1:]...)
				i--
			}
		} else if gv.PowerUps[i].refresh <= (gv.turnNumber/gv.ClientsInGame)-gv.PowerUps[i].lastRefresh {
			gv.PowerUps[i].lastRefresh = (gv.turnNumber / gv.ClientsInGame)
			if len(gv.PowerUps[i].possibleWeapons) > 0 {
				// add a random one from the possible
				toadd := rand.Intn(len(gv.PowerUps[i].possibleWeapons))
				gv.PowerUps[i].weapons = gv.PowerUps[i].possibleWeapons[toadd : toadd+1]
				fmt.Printf("Updated PowerUp at %d,%d to have %q\n", gv.PowerUps[i].pos.x, gv.PowerUps[i].pos.y, gv.PowerUps[i].weapons[0].name)
			}
		}
	}
	return false
}

func getPowerup(x, y int16, gv *GameVariables) (pu *PowerUp, found bool) {
	for i := 0; i < len(gv.PowerUps); i++ {
		if gv.PowerUps[i].pos.x == x && gv.PowerUps[i].pos.y == y {
			return &gv.PowerUps[i], true
		}
	}
	return &PowerUp{}, false
}

// damage functions
// should only be called if the weapon has enough ammo, and the player has enough moves
func genericDamage(hx, hy, start_x, start_y, direction int16, multiplier float32, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	// Check bounds
	if hx < 0 || hx >= int16(len(gv.GameMap[0])) || hy < 0 || hy >= int16(len(gv.GameMap)) {
		return false
	}

	fmt.Printf("Shot from %d,%d to %d,%d with %s\n", start_x, start_y, hx, hy, w.name)

	if gv.GameMap[hy][hx].occupied == true {
		damagePlayer(hx, hy, int16(float32(w.playerDamageMult)*multiplier), u, gv)
	} else if w.tileDamageMult < 0 {
		createTile(hx, hy, w.tileDamageMult, u, gv)
	} else {
		damageTile(hx, hy, int16(float32(w.tileDamageMult)*multiplier), u, gv)
	}

	// Update for animation
	u.WeaponHits = append(u.WeaponHits, HitInfo{damageType: w.damageType, pos: Position{x: hx, y: hy}, fromPos: Position{x: start_x, y: start_y}})
	return true
}

func damageStraight(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	rand_max := w.randomAngle
	// Add some random to the shots
	if rand_max > 0 {
		direction = direction + int16(rand.Intn(rand_max)-(rand_max/2))
	}
	// ray trace till we hit something
	hx, hy := traceDir(start_x, start_y, direction, w.distance, true, (w.tileDamageMult < 0), gv)

	return genericDamage(hx, hy, start_x, start_y, direction, 1.0, w, u, gv)
}

func damageSpread(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	num_shots := w.shotsPerShot
	rand_max := w.randomAngle
	for i := 0; i < num_shots; i++ {
		new_direction := direction
		if rand_max > 0 {
			new_direction = direction + int16(rand.Intn(rand_max)-(rand_max/2))
		}
		hx, hy := traceDir(start_x, start_y, new_direction, w.distance, true, (w.tileDamageMult < 0), gv)
		genericDamage(hx, hy, start_x, start_y, direction, 1.0, w, u, gv)
	}
	return true
}

func damageMelee(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	var hx, hy int16
	for direction < 0 {
		direction += 360
	}
	direction = direction % 360
	if direction < 45 || direction > 315 {
		hx = start_x + w.distance
		hy = start_y
	} else if direction < 135 {
		hx = start_x
		hy = start_y + w.distance
	} else if direction < 225 {
		hx = start_x - w.distance
		hy = start_y
	} else {
		hx = start_x
		hy = start_y - w.distance
	}

	return genericDamage(hx, hy, start_x, start_y, direction, 1.0, w, u, gv)
}

func damageWall(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	var hx, hy int16
	var hsx, hsy int16
	var hitareas []int16
	for direction < 0 {
		direction += 360
	}
	direction = direction % 360
	if direction < 45 || direction > 315 {
		hx = start_x + w.distance
		hy = start_y
	} else if direction < 135 {
		hx = start_x
		hy = start_y + w.distance
	} else if direction < 225 {
		hx = start_x - w.distance
		hy = start_y
	} else {
		hx = start_x
		hy = start_y - w.distance
	}
	hitareas = make([]int16, 0, 2*w.shotsPerShot)
	if hx != start_x {
		for dy := -(w.shotsPerShot / 2); dy < (w.shotsPerShot/2)+1; dy++ {
			hsx, hsy = trace(start_x, start_y, hx, hy+int16(dy), true, (w.tileDamageMult < 0), gv)
			hitareas = append(hitareas, hsx)
			hitareas = append(hitareas, hsy)
		}
	} else {
		for dx := -(w.shotsPerShot / 2); dx < (w.shotsPerShot/2)+1; dx++ {
			hsx, hsy = trace(start_x, start_y, hx+int16(dx), hy, true, (w.tileDamageMult < 0), gv)
			hitareas = append(hitareas, hsx)
			hitareas = append(hitareas, hsy)
		}

	}
	for i := 0; i < len(hitareas); i += 2 {
		genericDamage(hitareas[i], hitareas[i+1], start_x, start_y, direction, 1.0, w, u, gv)
	}
	return true
}

func damageExplosion(start_x, start_y, direction int16, w Weapon, u *UpdateGroup, gv *GameVariables) bool {
	// Add some random to the shots
	rand_max := w.randomAngle
	if rand_max > 0 {
		direction = direction + int16(rand.Intn(rand_max)-(rand_max/2))
	}
	// ray trace till we hit something
	ex, ey := traceDir(start_x, start_y, direction, w.distance, true, true, gv)
	aoe := [][]float32{
		[]float32{0.0, 0.0, 0.0, 0.1, 0.0, 0.0, 0.0},
		[]float32{0.0, 0.0, 0.2, 0.2, 0.2, 0.0, 0.0},
		[]float32{0.0, 0.2, 0.3, 0.4, 0.3, 0.2, 0.0},
		[]float32{0.1, 0.2, 0.4, 1.0, 0.4, 0.2, 0.1},
		[]float32{0.0, 0.2, 0.3, 0.4, 0.3, 0.2, 0.0},
		[]float32{0.0, 0.0, 0.2, 0.2, 0.2, 0.0, 0.0},
		[]float32{0.0, 0.0, 0.0, 0.1, 0.0, 0.0, 0.0},
	}
	hitareas := make([]int16, 0, len(aoe)*len(aoe[0])*2)
	for i := 0; i < len(aoe[0]); i++ {
		for j := 0; j < len(aoe); j++ {
			if aoe[i][j] > 0 {
				hx, hy := trace(ex, ey, ex+int16(i-(len(aoe[0])/2)), ey+int16(j-(len(aoe)/2)), true, false, gv)
				if w.tileDamageMult < 0 {
					hitareas = append(hitareas, hx)
					hitareas = append(hitareas, hy)
					hitareas = append(hitareas, int16(i))
					hitareas = append(hitareas, int16(j))
				} else {
					genericDamage(hx, hy, ex, ey, direction, aoe[i][j], w, u, gv)
				}
			}
		}
	}
	if w.tileDamageMult < 0 {
		for i := 0; i < len(hitareas); i += 4 {
			genericDamage(hitareas[i], hitareas[i+1], ex, ey, direction, aoe[hitareas[i+2]][hitareas[i+3]], w, u, gv)
		}
	}
	return true
}

// Weapons

var pistol Weapon = Weapon{
	name:             "pistol",
	damage:           damageStraight,
	playerDamageMult: 21,
	tileDamageMult:   30,
	damageType:       "bullet",
	ammo:             -1,
	movesCost:        4,
	distance:         64,
	randomAngle:      9,
	shotsPerShot:     1,
}

var sniper Weapon = Weapon{
	name:             "sniper",
	damage:           damageStraight,
	playerDamageMult: 70,
	tileDamageMult:   20,
	damageType:       "bullet",
	ammo:             5,
	movesCost:        8,
	distance:         64,
	randomAngle:      1,
	shotsPerShot:     1,
}

var shovel Weapon = Weapon{
	name:             "shovel",
	damage:           damageMelee,
	playerDamageMult: 30,
	tileDamageMult:   60,
	damageType:       "melee",
	ammo:             -1,
	movesCost:        3,
	distance:         1,
	randomAngle:      0,
	shotsPerShot:     1,
}

var bazooka Weapon = Weapon{
	name:             "bazooka",
	damage:           damageExplosion,
	playerDamageMult: 60,
	tileDamageMult:   100,
	damageType:       "explosion",
	ammo:             1,
	movesCost:        9,
	distance:         45,
	randomAngle:      15,
	shotsPerShot:     1,
}

var suicide Weapon = Weapon{
	name:             "suicide",
	damage:           damageExplosion,
	playerDamageMult: 900,
	tileDamageMult:   900,
	damageType:       "explosion",
	ammo:             1,
	movesCost:        5,
	distance:         0,
	randomAngle:      0,
	shotsPerShot:     1,
}

var shotgun Weapon = Weapon{
	name:             "shotgun",
	damage:           damageSpread,
	playerDamageMult: 10,
	tileDamageMult:   6,
	damageType:       "bullet",
	ammo:             6,
	movesCost:        6,
	distance:         45,
	randomAngle:      30,
	shotsPerShot:     12,
}

var encase Weapon = Weapon{
	name:             "encase",
	damage:           damageExplosion,
	playerDamageMult: 0,
	tileDamageMult:   -10,
	damageType:       "wall",
	ammo:             1,
	movesCost:        10,
	distance:         20,
	randomAngle:      6,
	shotsPerShot:     1,
}

var eztrump Weapon = Weapon{
	name:             "eztrump",
	damage:           damageWall,
	playerDamageMult: 0,
	tileDamageMult:   -30,
	damageType:       "wall",
	ammo:             6,
	movesCost:        5,
	distance:         3,
	randomAngle:      0,
	shotsPerShot:     5,
}

var minecraft Weapon = Weapon{
	name:             "minecraft",
	damage:           damageMelee,
	playerDamageMult: 0,
	tileDamageMult:   -30,
	damageType:       "wall",
	ammo:             30,
	movesCost:        2,
	distance:         1,
	randomAngle:      0,
	shotsPerShot:     1,
}

var flag Weapon = Weapon{
	name:             "flag",
	damage:           damageMelee,
	playerDamageMult: 0,
	tileDamageMult:   100,
	damageType:       "melee",
	ammo:             -1,
	movesCost:        2,
	distance:         1,
	randomAngle:      0,
	shotsPerShot:     1,
}
