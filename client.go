package main

import (
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/gol/dist"
	"uk.ac.bris.cs/gameoflife/sdl"
)

func client(params gol.Params, clientPort, brokerAddr string) {
	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)
	events2 := make(chan gol.Event, 1000)

	lis, err := net.Listen("tcp", clientPort)
	handleError(err)
	defer lis.Close()

	rpc.Register(&dist.ClientState{events})
	go rpc.Accept(lis)

	params.Broker, err = rpc.Dial("tcp", brokerAddr)
	for err != nil {
		params.Broker, err = rpc.Dial("tcp", brokerAddr)
	}

	go func() {
		for {
			fmt.Println(<-events)
		}
	}()

	world := gol.ReadFile(params, events, keyPresses)
	var res dist.BrokerRes
	go params.Broker.Call(dist.Broker, dist.BrokerReq{world}, &res)
	// util.VisualiseBooleanMatrix(res.FinalState, params.ImageWidth, params.ImageHeight)
	sdl.Start(params, events2, keyPresses)
}
