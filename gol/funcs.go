package gol

import "uk.ac.bris.cs/gameoflife/util"

// CalculateAliveCells returns a Cell for each alive cell
func CalculateAliveCells(world [][]bool) []util.Cell {
	cells := []util.Cell{}
	for y, col := range world {
		for x, v := range col {
			if v {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}
	return cells
}

// FindAliveNeighbours counts the number of living neighbours around the cell at x, y
func FindAliveNeighbours(world [][]bool, x int, y int) int {
	aliveNeighbours := 0
	for _, i := range []int{-1, 0, 1} {
		for _, j := range []int{-1, 0, 1} {
			if i == 0 && j == 0 {
				continue
			}

			wy := len(world)
			wx := len(world[0])
			// Add wx to x to wrap around from negatives
			// We index [y, x] because that's the standard
			living := world[(y+i+wy)%wy][(x+j+wx)%wx]
			if living {
				aliveNeighbours++
			}
		}
	}
	return aliveNeighbours
}

// NewWorld creates a blank world according to the data in p
func NewWorld(p Params) [][]bool {
	world := make([][]bool, p.ImageHeight)
	for x := range world {
		world[x] = make([]bool, p.ImageWidth)
	}
	return world
}

// HandleError handles fatal errors
func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
