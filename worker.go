package main

import (
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/gol/dist"
)

func worker(workerPort string) {
	lis, err := net.Listen("tcp", workerPort)
	handleError(err)
	defer lis.Close()
	rpc.Register(&dist.WorkerState{})
	rpc.Accept(lis)
}
