package gol

import (
	"encoding/gob"
	"fmt"
)

// DefaultNetParams returns a set of "default" network parameters
func DefaultNetParams() NetParams {
	return NetParams{
		ClientAddr:  "127.0.0.1:8000",
		ClientPort:  ":8000",
		BrokerAddr:  "3.238.71.81:8000",
		BrokerPort:  ":8100",
		WorkerAddrs: []string{"127.0.0.1:8000"},
		WorkerPorts: []string{":8000"},
	}
}

// ReadFile starts the processing of Game of Life. It should initialise channels and goroutines.
func readFile(p Params, events chan<- Event, keyPresses <-chan rune) ([][]bool, IoChannels) {

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	ioFilename := make(chan string)
	ioOutput := make(chan uint8)

	c := IoChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: ioFilename,
		output:   ioOutput,
		input:    make(chan uint8),
	}

	go startIo(p, c)

	world := NewWorld(p)
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
	return world, c
}

// Run is an entrypoint for tests
func Run(p Params, events chan Event, keyPresses chan rune) {
	gob.Register(&AliveCellsCount{})
	gob.Register(&ImageOutputComplete{})
	gob.Register(&StateChange{})
	gob.Register(&CellFlipped{})
	gob.Register(&TurnComplete{})
	gob.Register(&FinalTurnComplete{})

	defaults := DefaultNetParams()
	go RunClient(p, defaults.ClientPort, defaults.BrokerAddr, events, keyPresses)
}
