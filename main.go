package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"net/rpc"
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

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		// 10,
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")

	flag.StringVar(
		&params.Role,
		"role",
		"client",
		"Specifies the role of this machine. Can be client, broker or worker. Defaults to client.")

	flag.StringVar(
		&ClientAddr,
		"client",
		"127.0.0.1:8040",
		"Specifies the address of the client, which will run the SDL controller.")

	flag.StringVar(
		&BrokerAddr,
		"broker",
		"127.0.0.1:8020",
		"Specifies the address of the broker, which will communicate between all machines.")

	var workers string
	flag.StringVar(
		&workers,
		"workers",
		"127.0.0.1:8030",
		"Specifies the list of worker machines. #-separated.")

	flag.Parse()
	WorkerAddrs = strings.Split(workers, "#")
	params.Workers = make([]*rpc.Client, len(WorkerAddrs))
	clientPort := ":" + strings.Split(ClientAddr, ":")[1]
	brokerPort := ":" + strings.Split(BrokerAddr, ":")[1]
	// Assumes all workers on same port
	workerPort := ":" + strings.Split(WorkerAddrs[0], ":")[1]

	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)

	gob.Register(&gol.AliveCellsCount{})
	gob.Register(&gol.ImageOutputComplete{})
	gob.Register(&gol.StateChange{})
	gob.Register(&gol.CellFlipped{})
	gob.Register(&gol.TurnComplete{})
	gob.Register(&gol.FinalTurnComplete{})

	// First, client listens and broker connnects to it.
	// Then broker listens and client connects backwards.
	// Finally, each worker listens and broker connects to all of them.
	if params.Role == "client" {
		client(params, clientPort, BrokerAddr)
	} else if params.Role == "broker" {
		broker(params, ClientAddr, brokerPort, WorkerAddrs)
	} else if params.Role == "worker" {
		worker(workerPort)
	}
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
