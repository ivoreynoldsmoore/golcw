package main

import (
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/gol/dist"
)

func broker(params gol.Params, clientAddr, brokerPort string, workerAddrs []string) {
	var err error
	params.Client, err = rpc.Dial("tcp", clientAddr)
	for err != nil {
		params.Client, err = rpc.Dial("tcp", clientAddr)
	}

	lis, err := net.Listen("tcp", brokerPort)
	handleError(err)
	defer lis.Close()

	for idx, worker := range workerAddrs {
		var err error
		params.Workers[idx], err = rpc.Dial("tcp", worker)
		for err != nil {
			params.Workers[idx], err = rpc.Dial("tcp", worker)
		}
	}

	rpc.Register(&dist.BrokerState{params})
	rpc.Accept(lis)
}
