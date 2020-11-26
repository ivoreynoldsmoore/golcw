package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"runtime"
	"strings"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/gol/dist"
	"uk.ac.bris.cs/gameoflife/util"
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
		16,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		16,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		10,
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

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	// First, client listens and broker connnects to it.
	// Then broker listens and client connects backwards.
	// Finally, each worker listens and broker connects to all of them.
	if params.Role == "client" {

		lis, err := net.Listen("tcp", clientPort)
		handleError(err)
		defer lis.Close()
		go rpc.Accept(lis)

		params.Broker, err = rpc.Dial("tcp", BrokerAddr)
		for err != nil {
			params.Broker, err = rpc.Dial("tcp", BrokerAddr)
		}

		world := gol.ReadFile(params, events, keyPresses)
		var res dist.BrokerRes
		params.Broker.Call(dist.Broker, dist.BrokerReq{world}, &res)
		util.VisualiseBooleanMatrix(res.FinalState, params.ImageWidth, params.ImageHeight)
		// sdl.Start(params, events, keyPresses)
	} else if params.Role == "broker" {

		var err error
		params.Client, err = rpc.Dial("tcp", ClientAddr)
		for err != nil {
			params.Client, err = rpc.Dial("tcp", ClientAddr)
		}

		lis, err := net.Listen("tcp", brokerPort)
		handleError(err)
		defer lis.Close()

		for idx, worker := range WorkerAddrs {
			var err error
			params.Workers[idx], err = rpc.Dial("tcp", worker)
			for err != nil {
				params.Workers[idx], err = rpc.Dial("tcp", worker)
			}
		}

		rpc.Register(&dist.BrokerState{params})
		rpc.Accept(lis)
	} else if params.Role == "worker" {

		lis, err := net.Listen("tcp", workerPort)
		handleError(err)
		defer lis.Close()
		rpc.Register(&dist.WorkerState{})
		rpc.Accept(lis)
	}
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
