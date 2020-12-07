package gol

import (
	"fmt"
	"net"
	"net/rpc"
)

// RunBroker initialises and runs the broker
func RunBroker(params Params, clientAddr, brokerPort string, workerAddrs []string) {
	stopper := make(chan bool)

	workers := make([]*rpc.Client, len(workerAddrs))
	for idx, worker := range workerAddrs {
		var err error
		workers[idx], err = rpc.Dial("tcp", worker)
		for err != nil {
			workers[idx], err = rpc.Dial("tcp", worker)
		}
	}

	// Workaround for tests
	// Tests require constant reconnecting and disconnecting client-broker
	for {
		err := (error)(nil)
		client, err := rpc.Dial("tcp", clientAddr)
		for err != nil {
			client, err = rpc.Dial("tcp", clientAddr)
		}
		fmt.Println("LOG: Connected to client")

		lis, err := net.Listen("tcp", brokerPort)
		HandleError(err)

		fmt.Println("LOG: Broker accepting requests")
		rpc.Register(&BrokerState{Stopper: stopper, Client: client, Workers: workers})
		go rpc.Accept(lis)

		// Break and exit if not restarting, recv via Stop procedure
		restart := <-stopper
		lis.Close()
		if !restart {
			break
		}
	}
}
