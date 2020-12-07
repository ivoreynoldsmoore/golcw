package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

// WorkerState holds all the information the worker needs
type WorkerState struct {
	Stopper chan struct{}
	World   [][]bool
	Offset  int
	Slice   int
	Height  int
	Width   int
}

// WorkerReq is the request type for the worker function
type WorkerReq struct {
	RowAbove []bool
	RowBelow []bool
	Turn     int
}

// WorkerRes is the result type for the worker function
type WorkerRes struct {
	World   [][]bool
	Flipped []util.Cell
}

// InitWorkerReq initWorkerReq
type InitWorkerReq struct {
	World  [][]bool
	Offset int
	Slice  int
	Height int
	Width  int
}

// InitWorkerRes initWorkerRes
type InitWorkerRes struct {
}

// StopWorkerReq stopWorkerReq
type StopWorkerReq struct {
}

// StopWorkerRes stopWorkerRes
type StopWorkerRes struct {
}

// InitWorker initialises the worker
func (ws *WorkerState) InitWorker(req InitWorkerReq, res *InitWorkerRes) (err error) {
	ws.World = req.World
	ws.Offset = req.Offset
	ws.Slice = req.Slice
	ws.Height = req.Height
	ws.Width = req.Width
	return nil
}

// Worker is a machine that takes some section of the image and processes work on it for a number of turns
func (ws *WorkerState) Worker(req WorkerReq, res *WorkerRes) (err error) {
	// Copy rows above and below into their correct position
	// Add height so negative mod works
	ws.World[(ws.Offset+ws.Height-1)%ws.Height] = req.RowBelow
	ws.World[(ws.Offset+ws.Height+ws.Slice)%ws.Height] = req.RowAbove
	nextWorld := newWorld(ws.Height, ws.Width)
	cells := make([]util.Cell, 0)

	for y := ws.Offset; y < ws.Offset+ws.Slice; y++ {
		for x := 0; x < ws.Width; x++ {
			val := ws.World[y][x]
			cell := util.Cell{X: x, Y: y}
			aliveNeighbors := FindAliveNeighbours(ws.World, x, y)

			if val && (aliveNeighbors == 2 || aliveNeighbors == 3) {
				nextWorld[y][x] = true
			} else if !val && aliveNeighbors == 3 {
				nextWorld[y][x] = true
				cells = append(cells, cell)
				// par.c.events <- CellFlipped{Cell: cell, CompletedTurns: par.turn}
			} else if val {
				// No need to change dead cells, all cells are dead by default
				nextWorld[y][x] = false
				cells = append(cells, cell)
				// par.c.events <- CellFlipped{Cell: cell, CompletedTurns: par.turn}
			}
		}
	}
	ws.World = nextWorld
	res.World = nextWorld
	res.Flipped = cells
	return nil
}

// StopWorker is called by the broker on each worker to stop it
func (ws *WorkerState) StopWorker(req StopWorkerReq, res *StopWorkerRes) (err error) {
	var signal struct{}
	ws.Stopper <- signal
	return nil
}

func newWorld(h, w int) [][]bool {
	world := make([][]bool, h)
	for x := range world {
		world[x] = make([]bool, w)
	}
	return world
}
