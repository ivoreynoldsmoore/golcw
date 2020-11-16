package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

func calculateAliveCells(p Params, world [][]bool) []util.Cell {
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

func findAliveNeighbours(world [][]bool, x int, y int) int {
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

// Perform one iteration of the game of life on the argument world
func executor(p Params, c distributorChannels, turns int, world [][]bool) [][]bool {
	nextWorld := newWorld(p)

	for y, col := range world {
		for x, val := range col {
			cell := util.Cell{X: x, Y: y}
			aliveNeighbors := findAliveNeighbours(world, x, y)

			if val && (aliveNeighbors == 2 || aliveNeighbors == 3) {
				nextWorld[y][x] = true
			} else if !val && aliveNeighbors == 3 {
				nextWorld[y][x] = true
				c.events <- CellFlipped{Cell: cell, CompletedTurns: turns}
			} else if val {
				// No need to change dead cells, all cells are dead by default
				nextWorld[y][x] = false
				c.events <- CellFlipped{Cell: cell, CompletedTurns: turns}
			}
		}
	}
	return nextWorld
}
