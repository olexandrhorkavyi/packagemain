package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type client struct {
	conn net.Conn
	name string
	room string
}

type room struct {
	members map[net.Addr]*client
}

var rooms map[string]*room

var usage = `
/nick <name>: get a name, or stay anonymous
/join <room>: join a room, if room doesn't exist the new room will be created
/say <msg>:   send message to everyone in a room
/quit:        disconnects from the chat server
`

func main() {
	rooms = make(map[string]*room)

	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("unable to start telnet server: %s", err.Error())
	}

	defer listener.Close()
	log.Printf("telnet server started on 0.0.0.0:8888")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %s", err.Error())
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	log.Printf("client %s connected", conn.RemoteAddr().String())

	c := &client{
		conn: conn,
		name: "anonymous",
	}

	// help message
	c.sendMsgToClient(`
Welcome to TelnetChat!
` + usage)

	go c.startSender()
}

func (c *client) startSender() {
loop:
	for {
		msg, err := c.readClientInput()
		if err != nil {
			log.Printf("unable to read client input: %s", err.Error())
			continue
		}

		msgArgs := strings.Split(msg, " ")

		switch msgArgs[0] {
		case "/nick":
			if len(msgArgs) < 2 {
				c.sendMsgToClient("usage: /nick <name>")
				break
			}

			c.name = strings.Join(msgArgs[1:len(msgArgs)], " ")
			c.sendMsgToClient(fmt.Sprintf("all right, I will call you %s", c.name))
			break
		case "/join":
			if len(msgArgs) < 2 {
				c.sendMsgToClient("usage: /join <room>")
				break
			}

			c.quitCurrentRoom()

			c.room = strings.Join(msgArgs[1:len(msgArgs)], " ")
			_, ok := rooms[c.room]
			if !ok {
				rooms[c.room] = &room{
					members: make(map[net.Addr]*client),
				}
			}

			rooms[c.room].announce(c, fmt.Sprintf("> %s joined the room", c.name))
			rooms[c.room].members[c.conn.RemoteAddr()] = c

			c.sendMsgToClient(fmt.Sprintf("welcome to %s", c.room))
			break
		case "/say":
			if len(msgArgs) < 2 {
				c.sendMsgToClient("usage: /say <msg>")
				break
			}

			if len(c.room) == 0 {
				c.sendMsgToClient("join a room first to send a message")
				break
			}

			rooms[c.room].announce(c, fmt.Sprintf("> %s says: %s", c.name, strings.Join(msgArgs[1:len(msgArgs)], " ")))
			break

		case "/quit":
			log.Printf("client %s left", c.conn.RemoteAddr())

			c.quitCurrentRoom()

			c.sendMsgToClient("Sad to see you go =(")
			c.conn.Close()
			break loop
		default:
			c.sendMsgToClient(usage)
		}
	}
}

func (c *client) quitCurrentRoom() {
	if len(c.room) > 0 {
		delete(rooms[c.room].members, c.conn.RemoteAddr())
	}
}

func (c *client) readClientInput() (string, error) {
	c.conn.Write([]byte(""))
	s, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.Trim(s, "\r\n"), nil
}

func (c *client) sendMsgToClient(msg string) {
	if _, err := c.conn.Write([]byte(msg + "\n")); err != nil {
		log.Printf("unable to send message to a client: %s", err.Error())
	}
}

func (r *room) announce(from *client, msg string) {
	for _, c := range r.members {
		if from.conn.RemoteAddr() != c.conn.RemoteAddr() {
			c.sendMsgToClient(msg)
		}
	}
}
