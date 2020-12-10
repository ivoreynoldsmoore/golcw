package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"runtime"
	"strings"

	"uk.ac.bris.cs/gameoflife/gol"
)

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params
	var ClientAddr string
	var BrokerAddr string
	var WorkerAddrs []string

	defaults := gol.DefaultNetParams()

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the hiimage.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image.")

	flag.IntVar(
		&params.Turns,
		"turns",
		// 10,
		10000000000,
		"Specify the number of turns to process.")

	var role string
	flag.StringVar(
		&role,
		"role",
		"client",
		"Specifies the role of this machine. Can be client, broker or worker.")

	flag.StringVar(
		&ClientAddr,
		"client",
		defaults.ClientAddr,
		"Specifies the address of the client, which will run the SDL controller.")

	flag.StringVar(
		&BrokerAddr,
		"broker",
		defaults.BrokerAddr,
		"Specifies the address of the broker, which will communicate between all machines.")

	var workersString string
	flag.StringVar(
		&workersString,
		"workers",
		defaults.WorkerAddrs[0],
		"Specifies the list of worker machines. Space separated.")

	flag.Parse()
	WorkerAddrs = strings.Split(workersString, " ")
	clientPort := ":" + strings.Split(ClientAddr, ":")[1]
	brokerPort := ":" + strings.Split(BrokerAddr, ":")[1]
	// Assumes all workers on same port
	workerPort := ":" + strings.Split(WorkerAddrs[0], ":")[1]

	gob.Register(&gol.AliveCellsCount{})
	gob.Register(&gol.ImageOutputComplete{})
	gob.Register(&gol.StateChange{})
	gob.Register(&gol.CellFlipped{})
	gob.Register(&gol.TurnComplete{})
	gob.Register(&gol.FinalTurnComplete{})

	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	// First, client listens and broker connnects to it.
	// Then broker listens and client connects backwards.
	// Finally, each worker listens and broker connects to all of them.
	if role == "client" {
		go gol.RunClient(params, clientPort, BrokerAddr, events, keyPresses)
		// sdl.Start(params, events, keyPresses)
		for range events {
		}
	} else if role == "broker" {
		gol.RunBroker(params, clientPort, brokerPort, WorkerAddrs)
	} else if role == "worker" {
		gol.RunWorker(workerPort)
	}
}
