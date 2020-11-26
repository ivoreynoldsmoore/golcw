package dist

import (
	"bytes"
	"encoding/gob"

	"uk.ac.bris.cs/gameoflife/gol"
)

// ClientState holds all information the client needs
type ClientState struct {
	Events chan<- gol.Event
}

// ClientReq holds any data sent to the client
type ClientReq struct {
	// Would represent a []gol.Event, but we need to gob-ify it
	Events [][]byte
}

// ClientRes is the result type for the client function
type ClientRes struct {
}

// SendEvents is called from broker to send events back to the client
func (cs *ClientState) SendEvents(req ClientReq, res *ClientRes) (err error) {
	var buf bytes.Buffer
	dec := gob.NewDecoder(&buf)

	for _, e := range req.Events {
		var event gol.Event
		buf.Write(e)
		err := dec.Decode(&event)
		if err != nil {
			panic(err)
		}
		buf.Reset()
		cs.Events <- event
	}
	return nil
}

// EncodeEvent encodes an event in a gob encoding so that it can be send via RPC
func EncodeEvent(event gol.Event) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&event)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
