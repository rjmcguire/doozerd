// package doozer provides a github.com/ha/doozer compatible API to
// interract with persitence clients.
package doozerl

import (
	"github.com/4ad/doozer"
)

type Conn struct {
	*doozer.Conn
	Id	string
}

func Dial(addr string) (*Conn, error) {
	conn, err := doozer.Dial(addr)
	return &Conn{Conn: conn, Id: "1"}, err
}

func DialUri(uri, buri string) (*Conn, error) {
	conn, err := doozer.DialUri(uri, buri)
	return &Conn{Conn: conn, Id: "1"}, err
}

func (c *Conn) Set(path string, oldRev int64, body []byte) (rev int64, err error) {
	rev, err = c.Conn.Set(path, oldRev, body)
	if err != nil {
		return
	}

	ev, err := c.Wait("/ctl/persistence/"+c.Id+path, rev)
	if !ev.IsSet() {
		panic("set event was recorded as delete.")
	}
	return
}

func (c *Conn) Del(path string, rev int64) (err error) {
	err = c.Conn.Del(path, rev)
	if err != nil {
		return
	}

	ev, err := c.Conn.Wait(path, rev)
	if err != nil {
		return
	}

	ev, err = c.Conn.Wait("/ctl/persistence/"+c.Id+path, ev.Rev)
	if !ev.IsDel() {
		panic("delete event was recorded as set.")
	}
	return
}
