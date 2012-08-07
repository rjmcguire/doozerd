package main

import (
	"fmt"
	"github.com/ha/doozer"
	st "github.com/ha/doozerd/store"
	"regexp"
)

var exclude = regexp.MustCompile("^/ctl") // private doozer stuff.

// encode takes the binary representation of an operation
// and returns the human readable form.
func encode(ev doozer.Event) (mut string) {
	switch {
	case ev.IsSet():
		mut = st.MustEncodeSet(ev.Path, string(ev.Body), ev.Rev)
	case ev.IsDel():
		mut = st.MustEncodeDel(ev.Path, ev.Rev)
	}
	return
}

// monitor listens for new doozer events and sends those
// events to be saved by Store.
func monitor() {
	var (
		ev  doozer.Event
		err error
		rev int64 = -1
	)
	for {
		ev, err = conn.Wait("/**", rev)
		if err != nil {
			fatal(err)
		}
		rev = ev.Rev + 1
		// BUG(aram): what are these, NOPs?
		if ev.Flag == 0 {
			continue
		}

		if exclude.MatchString(ev.Path) {
			continue
		}

		mut := encode(ev)
		if *v {
			fmt.Println(mut)
		}

		store <- &mutation{ev: ev, mut: mut}
	}
}

// Notify acknowledges mutations by mirroring the affected
// namespace into /ctl/persistence/<id>.
func Notify() {
	prefix := "/ctl/persistence/" + fmt.Sprint(id)

	for m := range notify {
		ev := m.ev
		rev, err := conn.Rev()
		if err != nil {
			fatal(err)
		}
		switch {
		case ev.IsSet():
			body := []byte(fmt.Sprint(ev.Rev))
			_, err = conn.Set(prefix+ev.Path, rev, body)
		case ev.IsDel():
			err = conn.Del(prefix+ev.Path, rev)
		}
		if err != nil {
			fatal(err)
		}
	}
}
