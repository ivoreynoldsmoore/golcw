package gol

import (
	"fmt"
	"net/rpc"
)

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
	Role        string
	Client      *rpc.Client
	Broker      *rpc.Client
	Workers     []*rpc.Client
}

// ReadFile starts the processing of Game of Life. It should initialise channels and goroutines.
func ReadFile(p Params, events chan<- Event, keyPresses <-chan rune) [][]bool {

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	ioFilename := make(chan string)
	ioOutput := make(chan uint8)

	c := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: ioFilename,
		output:   ioOutput,
		input:    make(chan uint8),
	}

	go startIo(p, c)

	world := newWorld(p)
	c.command <- ioInput
	ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.input != 0
			// // If it's true, we've flipped it
			// if world[y][x] {
			// 	c.events <- CellFlipped{Cell: util.Cell{X: x, Y: y}, CompletedTurns: 0}
			// }
		}
	}
	return world
}
