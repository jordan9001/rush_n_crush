package rush_n_crush

import (
	"bytes"
	"encoding/json"
	"fmt"
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
)

type Position struct {
	x int16
	y int16
}

func (p Position) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"x\":")
	buf.WriteString(strconv.FormatUint(uint64(p.x), 10))
	buf.WriteString(",\"y\":")
	buf.WriteString(strconv.FormatUint(uint64(p.y), 10))
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type Tile struct {
	pos      Position
	tType    int8
	health   int16
	nextType int8
}

func (t Tile) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("{\"pos\":")
	posJSON, _ := t.pos.MarshalJSON()
	buf.Write(posJSON)
	buf.WriteString(",\"tType\":")
	buf.WriteString(strconv.FormatInt(int64(t.tType), 10))
	buf.WriteString(",\"health\":")
	buf.WriteString(strconv.FormatInt(int64(t.health), 10))
	buf.WriteString(",\"nextType\":")
	buf.WriteString(strconv.FormatInt(int64(t.nextType), 10))
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

var GameMap [][]Tile

func getRandomPosition() Position {
	for i := 0; i < 100; i++ {
		x := rand.Intn(len(GameMap[0]))
		y := rand.Intn(len(GameMap))
		if GameMap[y][x].tType == T_WALK {
			return Position{x: int16(x), y: int16(y)}
		}
	}
	return Position{x: -1, y: -1}
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
			tile.pos = Position{j, i}
			tile.tType = int8(t)
			row[j] = tile
			fmt.Printf("%d", t)
		}
		fmt.Printf("\n")
		GameMap[i] = row
	}
	return nil
}

func SendMap(id int8) {
	sendable, err := json.Marshal(GameMap)
	if err != nil {
		fmt.Printf("Got err : %q\n", err)
	}
	// Send the map
	Clients[id].ConWrite <- sendable
}
