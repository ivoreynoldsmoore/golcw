package gol

import (
	"fmt"
	"net"
	"net/rpc"
)

// RunBroker initialises and runs the broker
func RunBroker(params Params, clientAddr, brokerPort string, workerAddrs []string) {
	var err error
	client, err := rpc.Dial("tcp", clientAddr)
	for err != nil {
		client, err = rpc.Dial("tcp", clientAddr)
	}

	lis, err := net.Listen("tcp", brokerPort)
	HandleError(err)
	defer lis.Close()

	workers := make([]*rpc.Client, len(workerAddrs))
	for idx, worker := range workerAddrs {
		var err error
		workers[idx], err = rpc.Dial("tcp", worker)
		for err != nil {
			workers[idx], err = rpc.Dial("tcp", worker)
		}
	}

	fmt.Println("LOG: Broker accepting requests")
	rpc.Register(&BrokerState{Client: client, Workers: workers})
	rpc.Accept(lis)
}
