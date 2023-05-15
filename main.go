package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/novalagung/gubrak/v2"
	"io/ioutil"
	"log"
	"strings"
)

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

type M map[string]interface{}

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var connections = make([]*WebSocketConnection, 0)

var (
	upgrader = websocket.Upgrader{}
)

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		eachConn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *WebSocketConnection) bool {
		return each == currentConn
	}).Result()
	connections = filtered.([]*WebSocketConnection)
}

func handleIO(currentConn *WebSocketConnection, connection []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)
				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func main() {
	e := echo.New()
	e.GET("/", func(ctx echo.Context) error {
		content, err := ioutil.ReadFile("index.html")
		if err != nil {
			return ctx.String(400, "failed load file")
		}

		//return ctx.String(200, string(content))
		fmt.Fprintf(ctx.Response(), "%s", content)
		return nil
	})

	e.GET("/ws", func(ctx echo.Context) error {
		conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)

		if err != nil {
			return err
		}

		username := ctx.QueryParams().Get("username")

		currentConn := WebSocketConnection{Conn: conn, Username: username}
		connections = append(connections, &currentConn)

		go handleIO(&currentConn, connections)
		return nil
	})

	e.Logger.Fatal(e.Start(":1323"))
}
