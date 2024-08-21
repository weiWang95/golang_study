package main

import (
	"bytes"
	"fmt"
)

func main() {
	a := NewArea()
	for i := 0; i < AREA_WIDTH; i++ {
		a.SetBlock(NewPos(i, 1), true)
	}
	a.SetBlock(NewPos(0, 1), false)
	a.SetBlock(NewPos(7, 1), false)

	a.SetBlock(NewPos(3, 2), true)
	a.SetBlock(NewPos(5, 2), true)

	a.InitSun()

	a.Inspect()

	pos := NewPos(4, 2)
	a.AddLight(pos)
	updateMap := map[string]Pos{pos.Id(): pos}
	a.Update(updateMap)

	a.Inspect()
	fmt.Println()

	pos = NewPos(0, 4)
	a.SetBlock(pos, true)
	updateMap = map[string]Pos{pos.Id(): pos}
	a.Update(updateMap)

	fmt.Println()
	a.Inspect()
}

const AREA_WIDTH = 8
const MAX_LUM = 4

type Pos struct {
	X int
	Z int
}

func NewPos(x, z int) Pos {
	return Pos{X: x, Z: z}
}

func (p Pos) Id() string {
	return fmt.Sprintf("%d-%d", p.X, p.Z)
}

func (p Pos) Add(pos Pos) Pos {
	return NewPos(p.X+pos.X, p.Z+pos.Z)
}

func (p Pos) AddX(x int) Pos {
	return p.Add(Pos{X: x})
}

func (p Pos) AddZ(z int) Pos {
	return p.Add(Pos{Z: z})
}

func (p Pos) Range(fn func(p Pos)) {
	fn(p.AddX(-1))
	fn(p.AddX(1))
	fn(p.AddZ(-1))
	fn(p.AddZ(1))
}

type Luminance uint8

const LUM_BLOCK uint8 = 0b00001111
const LUM_SUN uint8 = 0b11110000

func NewLuminance(sunLum, blockLum uint8) Luminance {
	return Luminance(sunLum<<4 | blockLum)
}

func (l Luminance) SunLum() uint8 {
	return uint8(l) >> 4
}

func (l Luminance) BlockLum() uint8 {
	return uint8(l) & LUM_BLOCK
}

func (l Luminance) SetSunLum(sunLum uint8) Luminance {
	return Luminance(uint8(l)&LUM_BLOCK | (sunLum << 4))
}

func (l Luminance) SetBlockLum(blockLum uint8) Luminance {
	return Luminance(uint8(l)&LUM_SUN | blockLum)
}

func (l Luminance) CurLum() uint8 {
	if l.SunLum() > l.BlockLum() {
		return l.SunLum()
	}

	return l.BlockLum()
}

func MaxLum(l1, l2 Luminance) Luminance {
	var max Luminance
	if l1.SunLum() > l2.SunLum() {
		max = max.SetSunLum(l1.SunLum())
	} else {
		max = max.SetSunLum(l2.SunLum())
	}

	if l1.BlockLum() > l2.BlockLum() {
		max = max.SetBlockLum(l1.BlockLum())
	} else {
		max = max.SetBlockLum(l2.BlockLum())
	}

	return max
}

type Area struct {
	block [AREA_WIDTH][AREA_WIDTH]bool
	lum   [AREA_WIDTH][AREA_WIDTH]Luminance
	light map[string]interface{}
}

func NewArea() *Area {
	a := new(Area)
	a.light = make(map[string]interface{})
	return a
}

func (a *Area) InitSun() {
	updateMpa := make(map[string]Pos, AREA_WIDTH*AREA_WIDTH)
	for x := 0; x < AREA_WIDTH; x++ {
		var beat bool
		for z := 0; z < AREA_WIDTH; z++ {
			pos := NewPos(x, z)
			updateMpa[pos.Id()] = pos

			if a.IsBlock(pos) {
				beat = true
			}

			if !beat {
				a.SetLum(pos, a.GetLum(pos).SetSunLum(MAX_LUM))
			}
		}
	}

	a.Update(updateMpa)
}

func (a *Area) Update(updateMap map[string]Pos) {
	for i := 0; i < 15; i++ {
		fmt.Printf("times:%d len:%d\n", i, len(updateMap))
		newMap := make(map[string]Pos)

		for _, p := range updateMap {
			for _, up := range a.updateLum(p) {
				newMap[up.Id()] = up
			}
		}

		// a.Inspect()

		if len(newMap) == 0 {
			break
		}

		updateMap = newMap
	}
}

func (a *Area) updateLum(pos Pos) []Pos {
	if a.CheckOutRange(pos) {
		return nil
	}

	curLum := a.GetLum(pos)
	oldLum := curLum

	updates := make([]Pos, 0)
	if a.IsBlock(pos) && !a.HasLight(pos) {
		if curLum.SunLum() == MAX_LUM {
			updates = append(updates, a.CoverSun(pos)...)
		}
		curLum = NewLuminance(0, 0)
		a.SetLum(pos, curLum)
	} else {
		max := a.MaxLum(pos)
		maxBlock, maxSun := max.BlockLum(), max.SunLum()
		if !a.HasLight(pos) && maxBlock > 0 {
			curLum = curLum.SetBlockLum(maxBlock - 1)
		}
		if !a.IsSunBeat(pos) {
			if maxSun > 0 {
				curLum = curLum.SetSunLum(maxSun - 1)
			} else {
				curLum = curLum.SetSunLum(0)
			}
		}
		a.SetLum(pos, curLum)

		if !a.HasLight(pos) && oldLum == curLum {
			return nil
		}
	}

	pos.Range(func(p Pos) {
		if a.needUpdate(p, curLum) {
			updates = append(updates, p)
		}
	})

	return updates
}

func (a *Area) CoverSun(pos Pos) []Pos {
	arr := make([]Pos, 0)
	for {
		pos = pos.AddZ(1)

		if a.IsBlock(pos) {
			break
		}

		a.SetLum(pos, a.GetLum(pos).SetSunLum(0))
		arr = append(arr, pos)
	}

	return arr
}

func (a *Area) needUpdate(pos Pos, lum Luminance) bool {
	if a.CheckOutRange(pos) {
		return false
	}

	if a.IsBlock(pos) {
		return false
	}

	cur := a.GetLum(pos)

	return cur.BlockLum() != lum.BlockLum() || (cur.SunLum() != lum.SunLum() || cur.SunLum() != 0)
}

func (a *Area) SetBlock(pos Pos, exist bool) {
	if a.CheckOutRange(pos) {
		return
	}

	a.block[pos.X][pos.Z] = exist
}

func (a *Area) IsBlock(pos Pos) bool {
	if a.CheckOutRange(pos) {
		return true
	}

	return a.block[pos.X][pos.Z]
}

func (a *Area) AddLight(pos Pos) {
	a.light[pos.Id()] = nil
	a.SetLum(pos, MAX_LUM)
}

func (a *Area) HasLight(pos Pos) bool {
	_, ok := a.light[pos.Id()]
	return ok
}

func (a *Area) IsSunBeat(pos Pos) bool {
	sun := a.GetLum(pos).SunLum()
	return sun == MAX_LUM
}

func (a *Area) SetLum(pos Pos, lum Luminance) {
	if a.CheckOutRange(pos) {
		return
	}
	a.lum[pos.X][pos.Z] = lum
}

func (a *Area) GetLum(pos Pos) Luminance {
	if a.CheckOutRange(pos) {
		return 0
	}
	return a.lum[pos.X][pos.Z]
}

func (a *Area) MaxLum(pos Pos) Luminance {
	var max Luminance
	pos.Range(func(p Pos) {
		if a.IsBlock(p) {
			return
		}

		lum := a.GetLum(p)
		max = MaxLum(max, lum)
	})

	return max
}

func (a *Area) CheckOutRange(pos Pos) bool {
	return pos.X < 0 || pos.X >= AREA_WIDTH || pos.Z < 0 || pos.Z >= AREA_WIDTH
}

func (a *Area) Inspect() {
	for z := 0; z < AREA_WIDTH; z++ {
		var sunBuf bytes.Buffer
		var blockBuf bytes.Buffer
		var buf bytes.Buffer
		for x := 0; x < AREA_WIDTH; x++ {
			lum := a.lum[x][z]
			sun, block := lum.SunLum(), lum.BlockLum()
			if a.block[x][z] {
				sunBuf.WriteString(" ■ ")
				blockBuf.WriteString(" ■ ")
				buf.WriteString(" ■ ")
				// fmt.Print("■ ")
			} else {
				if sun == 0 {
					sunBuf.WriteString(" □ ")
				} else {
					sunBuf.WriteString(fmt.Sprintf("% 2d ", sun))
				}

				if block == 0 {
					blockBuf.WriteString(" □ ")
				} else {
					blockBuf.WriteString(fmt.Sprintf("% 2d ", block))
				}

				if lum.CurLum() == 0 {
					buf.WriteString(" □ ")
				} else {
					buf.WriteString(fmt.Sprintf("% 2d ", lum.CurLum()))
				}

				// fmt.Print(" □  ")
				// } else {
				// 	fmt.Printf("%d/%d ", lum.SunLum(), lum.BlockLum())
			}
		}
		fmt.Printf("%s  |  %s  |  %s\n", buf.String(), sunBuf.String(), blockBuf.String())
		// fmt.Println()
	}
}
