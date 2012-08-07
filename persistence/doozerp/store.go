package main

import (
	"github.com/ha/doozer"
)

// A mutation is the pair of the human readable and binary
// representation a a doozer operation.
type mutation struct {
	mut string
	ev  doozer.Event
}

// Store saves new mutations on disk.
func Store() {
	for m := range store {
		err := journal.WriteMutation(m.mut)
		if err != nil {
			fatal(err)
		}
		notify <- m
	}
}
