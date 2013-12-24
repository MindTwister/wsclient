package wsclient

import (
	"strings"
)

var rooms map[string]*Room = make(map[string]*Room)

type Room struct {
	name  string
	clients map[Client]bool
}

/*
GetRoom returns a room with the given name, if the room does not exist it will be automatically created with no Clients.
*/
func GetRoom(name string) *Room {
	lookupName := strings.ToLower(name)
	room, ok := rooms[lookupName]
	if ok {
		return room
	}
	r := Room{name, make(map[Client]bool)}
	rooms[lookupName] = &r
	return &r
}
func (r *Room) Broadcast(d interface{}) {
	for client := range r.clients {
		go func() {
			client.Send(d)
		}()
	}
}

func (r *Room) Name() string {
	return r.name
}

func (r *Room) Clients() []Client {
	out := make([]Client,0, len(r.clients))
	for client := range r.clients {
		out = append(out, client)
	}
	return out
}

func (r *Room) addClient(c Client) {
	r.clients[c]=true
}

func (r *Room) removeClient(c Client) {
	delete(r.clients, c)
}
