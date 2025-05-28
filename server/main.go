package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type Player struct {
	Username string `json:"username"`
	Password string `json:"password"`
	EXP      int    `json:"exp"`
	Level    int    `json:"level"`
}

var players []Player
var activeUsers = make(map[string]bool)
var connMutex sync.Mutex

func loadPlayers() {
	file, err := os.ReadFile("data/players.json")
	if err != nil {
		fmt.Println("Error reading players.json:", err)
		os.Exit(1)
	}
	err = json.Unmarshal(file, &players)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		os.Exit(1)
	}
}

func authenticate(username, password string) *Player {
	for _, p := range players {
		if p.Username == username && p.Password == password {
			return &p
		}
	}
	return nil
}

func handleClient(conn net.Conn, id int, authChan chan *Player, connChan chan net.Conn) {
	defer func() {
		conn.Close()
		// Send nil to indicate this connection failed
		select {
		case authChan <- nil:
		default:
		}
	}()

	var buf [512]byte
	n, err := conn.Read(buf[0:])
	if err != nil {
		fmt.Printf("Error reading from client %d: %v\n", id, err)
		return
	}

	input := string(buf[:n])
	var creds map[string]string
	err = json.Unmarshal([]byte(input), &creds)
	if err != nil {
		fmt.Fprintln(conn, "Invalid login format")
		return
	}

	username := creds["username"]
	password := creds["password"]

	if activeUsers[username] {
		fmt.Fprintln(conn, "Account already logged in")
		return
	}

	user := authenticate(username, password)
	if user == nil {
		fmt.Fprintln(conn, "invalid username/password")
		return
	}

	activeUsers[username] = true
	fmt.Fprintln(conn, "Login success")

	// Send the authenticated player and connection
	authChan <- user
	connChan <- conn
}

func safeWrite(conn net.Conn, data []byte) error {
	connMutex.Lock()
	defer connMutex.Unlock()

	// Set write deadline to prevent hanging
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Write(data)
	return err
}

func main() {
	loadPlayers()
	fmt.Println("Server is running on port 9000...")

	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer ln.Close()

	newConnChan := make(chan net.Conn)
	authChan := make(chan *Player)
	validConnChan := make(chan net.Conn)

	clients := make([]net.Conn, 2)
	users := make([]*Player, 2)
	connected := 0

	// Accept connections
	go func() {
		for {
			conn, err := ln.Accept()
			if err == nil {
				fmt.Println("Client connected.")
				newConnChan <- conn
			}
		}
	}()

	var timeout <-chan time.Time

	for connected < 2 {
		select {
		case conn := <-newConnChan:
			go handleClient(conn, connected, authChan, validConnChan)

		case player := <-authChan:
			if player != nil {
				// Wait for the corresponding connection
				select {
				case validConn := <-validConnChan:
					users[connected] = player
					clients[connected] = validConn
					fmt.Printf("Player %s authenticated.\n", player.Username)
					connected++

					if connected == 1 {
						fmt.Println("🕓 Waiting up to 30s for second player to join...")
						timeout = time.After(30 * time.Second)
					}
				case <-time.After(2 * time.Second):
					fmt.Println("Failed to get connection for authenticated player")
				}
			}

		case <-timeout:
			fmt.Println("⏰ Timeout: Second player did not join.")
			if clients[0] != nil {
				safeWrite(clients[0], []byte("Timeout waiting for second player.\n"))
				clients[0].Close()
			}
			return
		}
	}

	fmt.Println("✅ Both players authenticated! Starting game...")

	// Create safe wrapper connections
	safeClients := make([]net.Conn, 2)
	for i := 0; i < 2; i++ {
		safeClients[i] = clients[i]
	}

	StartGame(users, safeClients)

	// Clean up active users
	for _, user := range users {
		if user != nil {
			delete(activeUsers, user.Username)
		}
	}
}
