package wsclient

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"net/http"
	"reflect"
	"errors"
	"fmt"
)
const (
	MSG_JOIN      = "join"
	MSG_BROADCAST = "broadcast"
)

type JoinData struct {
	Room string
	Name string
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
	rooms map[*Room]bool
}

type Client interface {
	AddErrorHandler(func(error))
	Send(interface{})
	Receive() (MsgType, []byte)
	Name() string
	SetName(string)
	Join(*Room)
	Leave(*Room)
}

func(c *client) Join(r *Room) {
	r.addClient(c)
	c.rooms[r] = true
}

func(c *client) Leave(r *Room) {
	r.removeClient(c)
	delete(c.rooms,r)
}

func NewClient(name string, ws *websocket.Conn) Client {
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

func (c *client) Send(data interface{}) {
	err := websocket.JSON.Send(c.socket, data)
	if err != nil {
		c.err <- err
	}
}

func (c *client) Receive() (MsgType, []byte) {
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
	var c Client = NewClient("", ws)
	var room *Room
	c.AddErrorHandler(func(err error) {
		if room != nil {
			room.removeClient(c)
		}
	})
	for {
		msgType, byteData := c.Receive()
		switch msgType {
		case MSG_JOIN:
			jd := new(JoinData)
			json.Unmarshal(byteData, jd)
			c.SetName(jd.Name)
			room = GetRoom(jd.Room)
			c.Join(room)
		default:
			f, ok := registeredFunctions[msgType]
			if ok {
				f(c, byteData)
			}
		}
	}
}

var registeredFunctions map[MsgType]func(Client, []byte) = make(map[MsgType]func(Client, []byte))

func Register(msgType string, f interface{}) error {
	fInfo := reflect.TypeOf(f)
	if fInfo.Kind() != reflect.Func {
		return errors.New("The second argument MUST be function")
	}
	if fInfo.NumIn() != 2 {
		return errors.New("The second argument MUST be function with 2 arguments")
	}
	if fInfo.In(0) != reflect.TypeOf(NewClient).Out(0) {
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

func GetHandler() http.Handler {
	return websocket.Handler(connHandler)
}
