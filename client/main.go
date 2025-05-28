package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("❌ Failed to connect to server:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		fmt.Print("Enter password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		creds := map[string]string{
			"username": username,
			"password": password,
		}
		credJSON, _ := json.Marshal(creds)
		conn.Write(credJSON)

		reply := make([]byte, 256)
		n, err := conn.Read(reply)
		if err != nil {
			fmt.Println("❌ Error reading from server:", err)
			break
		}

		response := strings.TrimSpace(string(reply[:n]))
		fmt.Println("✅ Server response:", response)

		if response == "Login success" {
			playGame(conn, reader)
			break
		} else {
			fmt.Println("🔁 Try again or press Ctrl+C to quit.")
		}
	}
}

func playGame(conn net.Conn, reader *bufio.Reader) {
	serverReader := bufio.NewReader(conn)
	fmt.Println("🎮 Entering game mode...")

	for {
		// Read message from server
		message, err := serverReader.ReadString('\n')
		if err != nil {
			// Check if it's actually EOF or connection closed
			if err.Error() == "EOF" {
				fmt.Println("🎮 Game ended - Server closed connection")
			} else {
				fmt.Println("❌ Server disconnected:", err)
			}
			break
		}

		message = strings.TrimSpace(message)
		fmt.Println(message)

		// Game over conditions
		if strings.Contains(message, "wins") || strings.Contains(message, "🏆") {
			fmt.Println("🎮 Game Over! Thanks for playing!")
			// Wait a bit to see if there are more messages
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			continue
		}

		// Timeout condition
		if strings.Contains(message, "Timeout waiting for second player") {
			fmt.Println("⏰ Game cancelled due to timeout")
			break
		}

		// Check if it's our turn to choose a troop
		if strings.Contains(strings.ToLower(message), "choose troop") {
			for {
				fmt.Print("➡ Your choice (1-3): ")
				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("❌ Error reading input:", err)
					return
				}

				choice := strings.TrimSpace(input)
				if choice == "1" || choice == "2" || choice == "3" {
					// Send choice to server
					_, err = conn.Write([]byte(choice + "\n"))
					if err != nil {
						fmt.Println("❌ Error sending choice to server:", err)
						return
					}
					break
				} else {
					fmt.Println("❌ Invalid choice! Please enter 1, 2, or 3")
				}
			}
		}
	}
}
