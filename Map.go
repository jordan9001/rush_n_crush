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
	T_EMPTY  int8 = 1
	T_SWALL  int8 = 2
	T_WWALL  int8 = 3
	T_SLOWV  int8 = 4
	T_SLOWH  int8 = 5
	T_WLOWV  int8 = 6
	T_WLOWH  int8 = 7
	T_WALK   int8 = 8
	T_BUTTON int8 = 9
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

var GameMap [][]Tile

func getRandomPosition() Position {
	for i := 0; i < 100; i++ {
		x := rand.Intn(len(GameMap[0]) - 1)
		y := rand.Intn(len(GameMap) - 1)
		if GameMap[y][x].tType == T_WALK {
			return Position{x: int16(x), y: int16(y)}
		}
	}
	return Position{x: -1, y: -1}
}

func damageTile(x, y, damage int16, u *UpdateGroup) {
	if GameMap[y][x].nextType != GameMap[y][x].tType {
		GameMap[y][x].health -= damage
		if GameMap[y][x].health <= 0 {
			GameMap[y][x].tType = GameMap[y][x].nextType
		}
		u.TileUpdates = append(u.TileUpdates, GameMap[y][x])
	}
}

func LoadMap(map_args string) error {
	maparr := strings.Split(map_args, ",")
	var w, h int16
	var t int64
	var err error
	fmt.Printf("%s %s\n", maparr[0], maparr[1])

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
	GameMap = make([][]Tile, h)
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
			}
			tile.occupied = false
			row[j] = tile
			fmt.Printf("%d", t)
		}
		fmt.Printf("\n")
		GameMap[i] = row
	}
	return nil
}

func SendMap(id int8) {
	data, err := json.Marshal(GameMap)
	if err != nil {
		fmt.Printf("Got err : %q\n", err)
	}
	// Send the map message
	m := Message{"map", data}
	sendable, _ := m.MarshalJSON()
	Clients[id].ConWrite <- sendable
}

func traceDir(px, py, angle int16, chanceCoverBlock bool) (int16, int16) {
	var rad_ang float64 = math.Pi * float64(angle) / 180
	sin := math.Sin(rad_ang)
	cos := math.Cos(rad_ang)
	var length int16 = int16(len(GameMap) + len(GameMap[0]))
	var ex int16 = int16(cos*float64(length)) + px
	var ey int16 = int16(sin*float64(length)) + py
	return trace(px, py, ex, ey, chanceCoverBlock)
}

const (
	chance_m float64 = 3
	chance_b float64 = 0.3
)

func trace(px, py, x, y int16, chanceCoverBlock bool) (int16, int16) {
	fmt.Printf("Trace from %d,%d to %d,%d: ", px, py, x, y)
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
	err := dx - dy
	dx *= 2
	dy *= 2

	for {
		fmt.Printf("%d,%d ", sx, sy)
		if sx < 0 || sx >= int16(len(GameMap[0])) || sy < 0 || sy >= int16(len(GameMap)) {
			fmt.Printf(": Out of Bounds\n")
			break
		}
		tile := GameMap[sy][sx]
		if sx == x && sy == y {
			fmt.Printf(": Reached Target\n")
			break
		} else if tile.occupied && !(sx == px && sy == py) {
			fmt.Printf(": Occupied\n")
			break
		} else if tile.tType == T_SWALL || tile.tType == T_WWALL || tile.tType == T_EMPTY {
			fmt.Printf(": Wall\n")
			break
		} else if chanceCoverBlock {
			if tile.tType == T_SLOWV || tile.tType == T_SLOWH || tile.tType == T_WLOWV || tile.tType == T_WLOWH {
				// we have a chance to hit here, depending on how close we are to the cover
				dist2 := float64(((x - px) * (x - px)) + ((y - py) * (y - py)))
				chancepass := (chance_m / dist2) + chance_b
				randval := rand.Float64()
				if randval > chancepass {
					fmt.Printf(": Cover\n")
					break
				}
			}
		}

		if err > 0 {
			sx = sx + dirx
			err = err - dy
		} else {
			sy = sy + diry
			err = err + dx
		}
	}
	return sx, sy
}
