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

	// Intermediary events channel that isn't closed
	// This works around 'send on closed channel' errors in repeated test runs
	events2 := make(chan Event)

	var broker *rpc.Client
	fmt.Println("LOG: Create new ClientState")
	rpc.Register(&ClientState{Events: events2, Broker: broker})
	go rpc.Accept(lis)

	broker, err = rpc.Dial("tcp", brokerAddr)
	for err != nil {
		broker, err = rpc.Dial("tcp", brokerAddr)
	}

	stop := make(chan struct{}, 10)
	// Relay Events from events2 to events
	// Prevent race-conditions involving closing
	go func() {
		for {
			select {
			case <-stop:
				close(events)
				return
			case e := <-events2:
				fmt.Println("GOT FROM EVENTS2")
				fmt.Println(e)
				events <- e
			}
		}
	}()

	world, c := readFile(params, events, keyPresses)
	var res BrokerRes
	fmt.Println("LOG: Client sending request")
	err = broker.Call(Broker, BrokerReq{InitialState: world, Params: params}, &res)
	fmt.Println("LOG: Got final state from broker!")
	HandleError(err)
	SaveWorld(res.FinalState, params, c)
	time.Sleep(1 * time.Second)

	var stopSignal struct{}
	stop <- stopSignal

	// Stop SDL/tests gracefully
	fmt.Println("LOG: Client done, returning")
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
