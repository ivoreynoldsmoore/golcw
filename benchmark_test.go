package main

import (
	"fmt"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

// BenchmarkGol will be used to create benchmark graphs for the game of life
// Its code is based on the TestGol function, but it does not validate the output
// This is because that can be done seperately by the TestGol test
func BenchmarkGol(b *testing.B) {
	p := gol.Params{
		Turns:       5,
		ImageHeight: 5120,
		ImageWidth:  5120,
	}
	for _, threads := range []int{6} {
		p.Threads = threads
		testName := fmt.Sprintf("BenchmarkGol_%d\n", p.Threads)
		b.Run(testName, func(bs *testing.B) {
			events := make(chan gol.Event)
			b.ResetTimer()
			gol.Run(p, events, nil)
			for range events {
			}
		})
	}
}
