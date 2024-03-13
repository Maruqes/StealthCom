package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// USERS
type User struct {
	connection *websocket.Conn
	username   string
}

var users = []*User{}

func store_new_user(username string, conn *websocket.Conn) {
	newUser := &User{conn, username}

	index := 0
	for index < len(users) && users[index].username < newUser.username {
		index++
	}
	users = append(users[:index], append([]*User{newUser}, users[index:]...)...)
}

func search_for_user(username string) *User {
	low := 0
	high := len(users) - 1

	for low <= high {
		mid := (low + high) / 2

		if users[mid].username == username {
			return users[mid]
		} else if users[mid].username < username {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return nil
}

// CHAT
type Chat struct {
	Username1 string
	Username2 string
	Id        string
}

var chats = []*Chat{}

func store_new_chat(username1 string, username2 string) {
	id := uuid.New().String()
	newChat := &Chat{username1, username2, id}

	index := 0
	for index < len(chats) && chats[index].Id < newChat.Id {
		index++
	}
	chats = append(chats[:index], append([]*Chat{newChat}, chats[index:]...)...)
}

func search_for_chat(id string) *Chat {
	low := 0
	high := len(chats) - 1

	for low <= high {
		mid := (low + high) / 2

		if chats[mid].Id == id {
			return chats[mid]
		} else if chats[mid].Id < id {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return nil
}

func main() {
	//gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	engine.SetTrustedProxies([]string{"*"})

	engine.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})

	engine.GET("/create_user", func(c *gin.Context) {

		if c.Query("username") == "" || c.Query("username") == "undefined" || c.Query("username") == "null" {
			c.JSON(http.StatusBadRequest, gin.H{"err:": "Username is required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		var user User
		user.connection = conn
		user.username = c.Query("username")

		store_new_user(user.username, user.connection)
		conn.WriteMessage(1, []byte(user.username))
		fmt.Println("User connected: ", user.username)
	})

	engine.POST("/send_msg_user", func(c *gin.Context) {
		var json struct {
			Username string `json:"username"`
			Message  string `json:"message"`
		}
		err := c.BindJSON(&json)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		for i := 0; i < len(users); i++ {
			if users[i].username == json.Username {
				fmt.Println("Sending message to: ", users[i].username)
				err := users[i].connection.WriteMessage(1, []byte(json.Message))
				if err != nil {
					fmt.Printf("Error sending message to %s: %s\n", users[i].username, err.Error())
				}
			}
		}
	})

	engine.GET("/send_msg_all", func(c *gin.Context) {

		for i := 0; i < len(users); i++ {
			err := users[i].connection.WriteMessage(1, []byte("msg"))
			if err != nil {
				fmt.Printf("Error sending message to %s: %s\n", users[i].username, err.Error())
			}
		}
	})

	engine.POST("/create_chat", func(c *gin.Context) {
		var json struct {
			Username1 string `json:"username1"`
			Username2 string `json:"username2"`
		}
		err := c.BindJSON(&json)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		store_new_chat(json.Username1, json.Username2)
		c.JSON(http.StatusOK, gin.H{"uuid": chats[len(chats)-1].Id})
	})

	engine.POST("/send_msg_chat", func(c *gin.Context) {
		var json struct {
			Id      string `json:"id"`
			Message string `json:"message"`
		}
		err := c.BindJSON(&json)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		chat := search_for_chat(json.Id)
		if chat == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chat not found"})
			return
		}

		user1 := search_for_user(chat.Username1)
		user2 := search_for_user(chat.Username2)

		if user1 == nil || user2 == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
			return
		}

		err = user1.connection.WriteMessage(1, []byte(json.Message))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = user2.connection.WriteMessage(1, []byte(json.Message))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

	})

	engine.Run(":8080")

}
