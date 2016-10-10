package rush_n_crush

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

const (
	T_EMPTY int8 = 1
	T_SWALL int8 = 2
	T_WWALL int8 = 3
	T_SLOWV int8 = 4
	T_SLOWH int8 = 5
	T_WLOWV int8 = 6
	T_WLOWH int8 = 7
	T_WALK  int8 = 8
	T_SPAWN int8 = 9
	T_FLAG  int8 = 10
)
const (
	T_SWALL_H int16 = 100
	T_WWALL_H int16 = 30
	T_SLOW_H  int16 = 60
	T_WLOW_H  int16 = 10
)

type Position struct {
	x int16
	y int16
}

func (p Position) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"x\":")
	buf.WriteString(strconv.FormatInt(int64(p.x), 10))
	buf.WriteString(",\"y\":")
	buf.WriteString(strconv.FormatInt(int64(p.y), 10))
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type Tile struct {
	pos      Position
	tType    int8
	health   int16
	nextType int8
	occupied bool
	powerups WeaponCache
}

func (t Tile) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"pos\":")
	pos, _ := t.pos.MarshalJSON()
	buf.Write(pos)
	buf.WriteString(",\"tType\":")
	buf.WriteString(strconv.FormatInt(int64(t.tType), 10))
	buf.WriteString(",\"health\":")
	buf.WriteString(strconv.FormatInt(int64(t.health), 10))
	buf.WriteString(",\"nextType\":")
	buf.WriteString(strconv.FormatInt(int64(t.nextType), 10))
	buf.WriteString(",\"powerups\":")
	p, _ := t.powerups.MarshalJSON()
	buf.Write(p)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

type Spawn struct {
	pos    Position
	client int
}

func getPositionClose(x, y int16, gv *GameVariables) Position {
	// try spot
	if gv.GameMap[y][x].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
		return Position{x: x, y: y}
	}
	var tx, ty, i int16
	for i = 1; i < 100; i++ {
		// Do x extremes
		for tx = x - i; tx < x+i; tx++ {
			ty = y - i
			if gv.GameMap[ty][tx].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
				return Position{x: tx, y: ty}
			}
			ty = y + i
			if gv.GameMap[ty][tx].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
				return Position{x: tx, y: ty}
			}
		}
		// Do y extremes
		for ty = y - (i - 1); ty < y+(i-1); ty++ {
			tx = x - i
			if gv.GameMap[ty][tx].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
				return Position{x: tx, y: ty}
			}
			tx = x + i
			if gv.GameMap[ty][tx].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
				return Position{x: tx, y: ty}
			}
		}
	}
	fmt.Printf("Error, could not get position\n")
	return Position{x: -1, y: -1}
}

func getRandomPosition(gv *GameVariables) Position {
	for i := 0; i < 100; i++ {
		x := rand.Intn(len(gv.GameMap[0]) - 1)
		y := rand.Intn(len(gv.GameMap) - 1)
		if gv.GameMap[y][x].tType >= T_WALK && gv.GameMap[y][x].occupied == false {
			return Position{x: int16(x), y: int16(y)}
		}
	}
	fmt.Printf("Error, could not get position\n")
	return Position{x: -1, y: -1}
}

func damageTile(x, y, damage int16, u *UpdateGroup, gv *GameVariables) {
	if gv.GameMap[y][x].nextType != gv.GameMap[y][x].tType {
		gv.GameMap[y][x].health -= damage
		if gv.GameMap[y][x].health <= 0 {
			gv.GameMap[y][x].tType = gv.GameMap[y][x].nextType
		}
		u.TileUpdates = append(u.TileUpdates, gv.GameMap[y][x])
	}
}

func LoadMap(map_args string, gv *GameVariables) error {
	maparr := strings.Split(map_args, ",")
	var w, h int16
	var t int64
	var err error

	t, err = strconv.ParseInt(maparr[0], 10, 16)
	if err != nil {
		fmt.Printf("Got err %q\n", err)
		return err
	}
	w = int16(t)

	t, err = strconv.ParseInt(maparr[1], 10, 16)
	if err != nil {
		fmt.Printf("Got err %q\n", err)
		return err
	}
	h = int16(t)

	fmt.Printf("Loading map of size %dx%d\n", w, h)

	// Allocate the map
	gv.GameMap = make([][]Tile, h)
	for i := int16(0); i < h; i++ {
		row := make([]Tile, w)
		for j := int16(0); j < w; j++ {
			t, err = strconv.ParseInt(maparr[(i*w)+j+2], 10, 8)
			if err != nil {
				fmt.Printf("Got err %q\n", err)
				return err
			}
			var tile Tile
			tile.tType = int8(t)
			tile.pos = Position{x: j, y: i}
			tile.nextType = tile.tType
			if tile.tType == T_SWALL {
				tile.nextType = T_WALK
				tile.health = T_SWALL_H
			} else if tile.tType == T_WWALL {
				tile.nextType = T_WALK
				tile.health = T_WWALL_H
			} else if tile.tType == T_SLOWV || tile.tType == T_SLOWH {
				tile.nextType = T_WALK
				tile.health = T_SLOW_H
			} else if tile.tType == T_WLOWV || tile.tType == T_WLOWH {
				tile.nextType = T_WALK
				tile.health = T_WLOW_H
			} else if tile.tType == T_SPAWN {
				// add a spawn
			}
			tile.occupied = false
			row[j] = tile
		}
		gv.GameMap[i] = row
	}
	printAsciiMap(gv.GameMap)
	return nil
}

func SendMap(id int, gv *GameVariables) {
	data, err := json.Marshal(gv.GameMap)
	if err != nil {
		fmt.Printf("Got err : %q\n", err)
	}
	// Send the map message
	m := Message{"map", data}
	sendable, _ := m.MarshalJSON()
	Clients[id].ConWrite <- sendable
}

func traceDir(px, py, angle, distance int16, chanceCoverBlock bool, stopBeforeHit bool, gv *GameVariables) (int16, int16) {
	var rad_ang float64 = math.Pi * float64(angle) / 180
	sin := math.Sin(rad_ang)
	cos := math.Cos(rad_ang)
	var ex int16 = int16(cos*float64(distance)) + px
	var ey int16 = int16(sin*float64(distance)) + py
	return trace(px, py, ex, ey, chanceCoverBlock, stopBeforeHit, gv)
}

// Chance to hit cover
const (
	CHANCE_M float64 = 6
	CHANCE_B float64 = 0.6
)

func trace(px, py, x, y int16, chanceCoverBlock bool, stopBeforeHit bool, gv *GameVariables) (int16, int16) {
	// fmt.Printf("Trace from %d,%d to %d,%d: ", px, py, x, y)
	var dx, dirx, dy, diry int16
	if x > px {
		dx = x - px
		dirx = 1
	} else {
		dx = px - x
		dirx = -1
	}
	if y > py {
		dy = y - py
		diry = 1
	} else {
		dy = py - y
		diry = -1
	}

	sx := px
	sy := py
	prevx := px
	prevy := py
	err := dx - dy
	dx *= 2
	dy *= 2

	for {
		if sx < 0 || sx >= int16(len(gv.GameMap[0])) || sy < 0 || sy >= int16(len(gv.GameMap)) {
			break
		}
		tile := gv.GameMap[sy][sx]
		if sx == x && sy == y {
			break
		} else if tile.occupied && !(sx == px && sy == py) {
			break
		} else if tile.tType == T_SWALL || tile.tType == T_WWALL || tile.tType == T_EMPTY {
			break
		} else if chanceCoverBlock {
			if tile.tType == T_SLOWV || tile.tType == T_SLOWH || tile.tType == T_WLOWV || tile.tType == T_WLOWH {
				// we have a chance to hit here, depending on how close we are to the cover
				dist2 := float64(((sx - px) * (sx - px)) + ((sy - py) * (sy - py)))
				chancepass := (CHANCE_M / dist2) + CHANCE_B
				randval := rand.Float64()
				if randval > chancepass {
					break
				}
			}
		}

		prevx = sx
		prevy = sy

		if err > 0 {
			sx = sx + dirx
			err = err - dy
		} else {
			sy = sy + diry
			err = err + dx
		}
	}
	if stopBeforeHit {
		return prevx, prevy
	}
	return sx, sy
}

func printAsciiMap(gm [][]Tile) {
	for y := 0; y < len(gm); y++ {
		for x := 0; x < len(gm[0]); x++ {
			if gm[y][x].tType == T_EMPTY {
				fmt.Printf("#")
			} else if gm[y][x].tType == T_SWALL {
				fmt.Printf("8")
			} else if gm[y][x].tType == T_WWALL {
				fmt.Printf("6")
			} else if gm[y][x].tType == T_SLOWV {
				fmt.Printf("|")
			} else if gm[y][x].tType == T_SLOWH {
				fmt.Printf("-")
			} else if gm[y][x].tType == T_WLOWV {
				fmt.Printf(";")
			} else if gm[y][x].tType == T_WLOWH {
				fmt.Printf("~")
			} else if gm[y][x].tType == T_WALK {
				fmt.Printf(" ")
			} else if gm[y][x].tType == T_SPAWN {
				fmt.Printf("+")
			} else if gm[y][x].tType == T_FLAG {
				fmt.Printf("*")
			}
		}
		fmt.Printf("\n")
	}
}
