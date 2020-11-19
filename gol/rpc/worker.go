package rpc

import "uk.ac.bris.cs/gameoflife/gol"

type WorkerState struct {
}

type WorkerReq struct {
	offset  int
	height  int
	width   int
	section [][]bool
	turn    int
}

type WorkerRes struct {
	nextSection [][]bool
}

// Each machine will be a worker that takes some section of the image and processes work on it for a number of turns
func (ws *WorkerState) worker(req WorkerReq, res *WorkerRes) (err error) {
	// Note that each req.subWorld is taller in each direction by 1 index compared to nextSubWorld
	// subWorld has height req.height + 2 and width req.width
	nextSection := make([][]bool, req.height)
	for idx := range nextSection {
		nextSection[idx] = make([]bool, req.width)
	}

	for y := 1; y <= req.height; y++ {
		for x := 0; x < req.width; x++ {
			val := req.section[y][x]
			// cell := util.Cell{X: x, Y: req.offset + y}
			aliveNeighbors := gol.FindAliveNeighbours(req.section, x, y)

			if val && (aliveNeighbors == 2 || aliveNeighbors == 3) {
				nextSection[y-1][x] = true
			} else if !val && aliveNeighbors == 3 {
				nextSection[y-1][x] = true
				// par.c.events <- CellFlipped{Cell: cell, CompletedTurns: par.turn}
			} else if val {
				// No need to change dead cells, all cells are dead by default
				nextSection[y][x] = false
				// par.c.events <- CellFlipped{Cell: cell, CompletedTurns: par.turn}
			}
		}
	}
	res.nextSection = nextSection
	return nil
}
