package wsclient

import (
	"code.google.com/p/go.net/websocket"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"testing"
	"time"
	"fmt"
)

// In the following time.Sleep is used multiple times, this is in order for server messages to propagate properly before testing the assertions.

var (
	wsPort = ":4567"
)

type Sample struct {
	A string
	B int
}

func init() {
	handlerFunc := GetHandler()
	http.Handle("/ws", handlerFunc)
	go (func() {
		http.ListenAndServe("localhost"+wsPort, nil)
	})()
}

type TestMessage struct {
	Type string
	Data interface{}
}

func TestCanCreateRoom(t *testing.T) {
	Convey("Given a room name", t, func() {
		roomName := "Test Room 1"
		Convey("A Room can be created", func() {
			room := GetRoom(roomName)
			So(room, ShouldNotBeNil)
			Convey("Its name can be retrieved", func() {
				So(roomName, ShouldEqual, room.Name())
			})
		})
	})
}
func TestRoomCanHaveUsers(t *testing.T) {
	Convey("Given a room", t, func() {
		room := GetRoom("Test Room 2")
		numUsers := len(room.Clients())
		user := NewClient("User 1", nil)
		Convey("If we add a user", func() {
			Convey("That users name can be retrieved", func(){
				So(user.Name(),ShouldEqual, "User 1")
			})
			room.addClient(user)
			Convey("The userlist length should be 1 greater than before", func() {
				So(len(room.Clients()), ShouldEqual, numUsers+1)
				fmt.Println(room.Clients())
				Convey("And user should contain the new user", func() {
					So(user, ShouldBeIn, room.Clients())
				})
			})
			Convey("Then we can remove the user again", func() {
				user.Leave(room)
				So(len(room.Clients()), ShouldEqual, numUsers)
			})
		})
	})
}

func TestSocketCanBeReached(t *testing.T) {
	Convey("Given that the webserver is runnning", t, func() {
		Convey("And we can connect to the server", func() {
			ws, err := websocket.Dial("ws://localhost"+wsPort+"/ws", "", "http://localhost"+wsPort+"/")
			So(err, ShouldBeNil)
			Convey("The server should let us join", func() {
				d := map[string]string{"Type": MSG_JOIN, "Room": "TestRoom1", "Name": "TestUserWebsocket"}
				ws.SetDeadline(time.Now().Add(time.Second * 3))
				websocket.JSON.Send(ws, d)
				room := GetRoom("TestRoom1")
				Convey("And the room should have at least one participant", func() {
					// Sleep to let the data reach the server
					time.Sleep(time.Millisecond * 50)
					So(len(room.Clients()), ShouldEqual, 1)
				})
				Convey("Afterwards we close up the socket", func() {
					ws.Close()
					time.Sleep(time.Millisecond)
					Convey("Now the number of users should be 0", func() {
						So(len(room.Clients()), ShouldEqual, 0)
					})
				})
			})
		})
	})
}

func TestSocketReceivesData(t *testing.T) {
	Convey("Given a webserver, a room and a user", t, func() {
		ws, err := websocket.Dial("ws://localhost"+wsPort+"/ws", "", "http://localhost"+wsPort+"/")
		So(err, ShouldBeNil)
		room := GetRoom("TestRoomReceive")
		joinMsg := map[string]string{"Type": MSG_JOIN, "Room": "TestRoomReceive", "Name": "TestUserReceive"}
		Convey("If a user joins the room", func() {
			websocket.JSON.Send(ws, joinMsg)
			Convey("That user should receive messages on broadcast", func() {
				time.Sleep(time.Millisecond)
				testMsg := map[string]string{"Type": MSG_BROADCAST, "Message": "Hi"}
				room.Broadcast(testMsg)
				rcvData := make(map[string]string)
				// Set the deadline so we don't wait indefinitely for response
				ws.SetReadDeadline(time.Now().Add(time.Millisecond * 50))
				websocket.JSON.Receive(ws, &rcvData)
				So(rcvData, ShouldResemble, testMsg)
			})
		})
	})
}

func TestRegisterForMessage(t *testing.T) {
	ws, _ := websocket.Dial("ws://localhost"+wsPort+"/ws", "", "http://localhost"+wsPort+"/")
	Convey("Given that the register function breaks on wrong arguments", t, func(){
		err := Register("throwaway", 8)
		So(err, ShouldNotBeNil) 
		err = Register("throwaway", func(){})
		So(err, ShouldNotBeNil) 
		err = Register("throwaway", func(c *Client){})
		So(err, ShouldNotBeNil) 
		err = Register("throwaway", func(c string, i int){})
		So(err, ShouldNotBeNil) 
		Convey("After registering for a specific type", func(){
			hasBeenCalledWithCorrectArguments := false
			err := Register("vote", func(c Client, s Sample){
				if s.A == "TestString" && s.B == 42 {
					hasBeenCalledWithCorrectArguments = true
				} else {
					fmt.Println(s)
				}
			})
			So(err, ShouldBeNil)
			Convey("And sending a message to the server", func(){
				websocket.JSON.Send(ws, map[string]interface{}{"Type":"vote","A":"TestString","B":42})
				Convey("The callback should have been reached with the correct arguments", func(){
					time.Sleep(time.Millisecond)
					So(hasBeenCalledWithCorrectArguments,ShouldEqual,true)
				})
			})
		})
	})
}
