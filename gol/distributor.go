package gol

import (
	"fmt"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events    chan<- Event
	ioCommand chan<- ioCommand
	ioIdle    <-chan bool
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	world := newWorld(p)
	cells := util.ReadAliveCells(fmt.Sprintf("images/%dx%d.pgm", p.ImageWidth, p.ImageHeight), p.ImageWidth, p.ImageHeight)
	for _, cell := range cells {
		world[cell.Y][cell.X] = true
		c.events <- CellFlipped{Cell: cell, CompletedTurns: 0}
	}

	turn := 0
	for turn = 0; turn < p.Turns; turn++ {
		world = executor(p, c, turn, world)
		// 1st turn completes when i = 0, etc.
		c.events <- TurnComplete{turn + 1}
	}
	c.events <- FinalTurnComplete{Alive: calculateAliveCells(p, world), CompletedTurns: i}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func newWorld(p Params) [][]bool {
	world := make([][]bool, p.ImageHeight)
	for x := range world {
		world[x] = make([]bool, p.ImageWidth)
	}
	return world
}
