// File: client/main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
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
			fmt.Println("Error reading from server:", err)
			break
		}

		response := strings.TrimSpace(string(reply[:n]))
		fmt.Println("\u2714 Server response:", response)

		if response == "Login success" {
			fmt.Println("\U0001f3ae Entering game mode...")
			playGame(conn, reader)
			break
		} else {
			fmt.Println("\u274C Try again or press Ctrl+C to quit.")
		}
	}
}

func playGame(conn net.Conn, reader *bufio.Reader) {
	serverReader := bufio.NewReader(conn)

	for {
		message, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("\u274C Server disconnected: EOF")
			break
		}

		fmt.Print(message)

		if strings.Contains(message, "wins") || strings.Contains(message, "\U0001f3c6") {
			break
		}

		if strings.Contains(message, "Choose troop") {
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "1" || input == "2" || input == "3" {
				if _, err := conn.Write([]byte(input + "\n")); err != nil {
					fmt.Println("\u274C Error sending input:", err)
					break
				}
			}
		}
	}
}
