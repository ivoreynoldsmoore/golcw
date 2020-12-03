package gol

import (
	"fmt"
	"net"
	"net/rpc"
	"time"
)

// RunClient is used an an entrypoint for tests and for the main program
func RunClient(params Params, clientPort, brokerAddr string, events chan Event, keyPresses chan rune) [][]bool {
	lis, err := net.Listen("tcp", clientPort)
	HandleError(err)
	defer lis.Close()

	var broker *rpc.Client
	rpc.Register(&ClientState{Events: events, Broker: broker})
	go rpc.Accept(lis)

	broker, err = rpc.Dial("tcp", brokerAddr)
	for err != nil {
		broker, err = rpc.Dial("tcp", brokerAddr)
	}

	world, c := readFile(params, events, keyPresses)
	var res BrokerRes
	fmt.Println("LOG: Client sending request")
	broker.Call(Broker, BrokerReq{InitialState: world, Params: params}, &res)
	SaveWorld(res.FinalState, params, c)

	fmt.Println("LOG: Client done, returning")
	time.Sleep(1 * time.Second)
	// Stop SDL/tests gracefully
	close(events)
	return res.FinalState
}

// SaveWorld takes a world and saves it as specified by the tests
func SaveWorld(world [][]bool, p Params, c ioChannels) {
	c.command <- ioOutput
	c.filename <- fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] {
				c.output <- 255
			} else {
				c.output <- 0
			}
		}
	}
}
