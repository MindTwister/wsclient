package wsclient

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

const (
	MSG_JOIN      = "join"
	MSG_BROADCAST = "broadcast"
)

type JoinData struct {
	Room string
	Name string
}

type BroadcastData struct {
	Room string
}

type Message struct {
	Type MsgType
}

type MsgType string

type client struct {
	socket        *websocket.Conn
	err           chan error
	errorHandlers []func(error)
	name          string
	rooms         map[*Room]bool
}

type Client interface {
	AddErrorHandler(func(error))
	Send(interface{})
	Name() string
	SetName(string)
	Join(*Room)
	Leave(*Room)
}

func (c *client) Join(r *Room) {
	r.addClient(c)
	c.rooms[r] = true
}

func (c *client) Leave(r *Room) {
	r.removeClient(c)
	delete(c.rooms, r)
}

func newClient(name string, ws *websocket.Conn) Client {
	c := new(client)
	c.name = name
	c.socket = ws
	c.rooms = make(map[*Room]bool)
	c.err = make(chan error)
	go (func() {
		err := <-c.err
		for _, h := range c.errorHandlers {
			h(err)
		}
	})()
	return c
}

func (c *client) SetName(n string) {
	c.name = n
}

func (c *client) Name() string {
	return c.name
}

// Send arbitrary data to the client, the data must be JSON serializable
func (c *client) Send(data interface{}) {
	err := websocket.JSON.Send(c.socket, data)
	if err != nil {
		c.err <- err
	}
}

func (c *client) receive() (MsgType, []byte) {
	var byteData []byte
	err := websocket.Message.Receive(c.socket, &byteData)
	if err != nil {
		c.err <- err
	}
	m := new(Message)
	json.Unmarshal(byteData, m)
	return m.Type, byteData
}

func (c *client) AddErrorHandler(f func(error)) {
	c.errorHandlers = append(c.errorHandlers, f)
}

func connHandler(ws *websocket.Conn) {
	var c Client = newClient("", ws)
	var room *Room
	c.AddErrorHandler(func(err error) {
		if room != nil {
			room.removeClient(c)
		}
	})
	for {
		msgType, byteData := c.(*client).receive()
		switch msgType {
		case MSG_JOIN:
			jd := new(JoinData)
			json.Unmarshal(byteData, jd)
			c.SetName(jd.Name)
			room = GetRoom(jd.Room)
			c.Join(room)
		case MSG_BROADCAST:
			bd := new(BroadcastData)
			json.Unmarshal(byteData, bd)
			data := make(map[string]interface{})
			json.Unmarshal(byteData, &data)
			GetRoom(bd.Room).Broadcast(data)
		default:
			fn, ok := registeredFunctions[msgType]
			if ok {
				fn(c, byteData)
			}
		}
	}
}

var registeredFunctions map[MsgType]func(Client, []byte) = make(map[MsgType]func(Client, []byte))

/*
Register takes 2 parameters, a MsgType, and a function which takes 2 parameters, a Client interface and a struct.

The registered function is called whenever the socket receives a message with a set Type field. eg. { Type : <MsgType>, ... }

The struct itself does not have to include the Type field
*/
func Register(msgType MsgType, f interface{}) error {
	fInfo := reflect.TypeOf(f)
	if fInfo.Kind() != reflect.Func {
		return errors.New("The second argument MUST be function")
	}
	if fInfo.NumIn() != 2 {
		return errors.New("The second argument MUST be function with 2 arguments")
	}
	if fInfo.In(0) != reflect.TypeOf(newClient).Out(0) {
		return errors.New(fmt.Sprintf("The first argument to the function MUST implement Client"))
	}
	callBackType := fInfo.In(1)
	callBackFunc := reflect.ValueOf(f)
	callBack := func(c Client, data []byte) {
		cbData := reflect.New(callBackType)
		cbDataInterface := cbData.Interface()
		json.Unmarshal(data, cbDataInterface)
		arguments := []reflect.Value{reflect.ValueOf(c), reflect.ValueOf(cbDataInterface).Elem()}
		callBackFunc.Call(arguments)
	}
	registeredFunctions[MsgType(msgType)] = callBack
	return nil
}

// GetHandler returns a http.Handler that can be bound to a path using http.Handle, this handler takes care websocket handling
func GetHandler() http.Handler {
	return websocket.Handler(connHandler)
}
