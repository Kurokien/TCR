package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Safe write function with error handling
func safeWriteToConn(conn net.Conn, data []byte, playerID int) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// Check if connection is still alive
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Write(data)
	if err != nil {
		fmt.Printf("⚠️ Error writing to player %d: %v\n", playerID, err)
		return err
	}
	return nil
}

// Broadcast message to all players safely
func safeBroadcast(conns []net.Conn, message []byte) {
	var wg sync.WaitGroup
	for i, conn := range conns {
		if conn != nil {
			wg.Add(1)
			go func(conn net.Conn, playerID int) {
				defer wg.Done()
				safeWriteToConn(conn, message, playerID+1)
			}(conn, i)
		}
	}
	wg.Wait()
}

func StartGame(players []*Player, conns []net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("‼ Panic in StartGame:", r)
		}
		// Clean close all connections
		for i, conn := range conns {
			if conn != nil {
				fmt.Printf("🔌 Closing connection for player %d\n", i+1)
				conn.Close()
			}
		}
	}()

	troops := LoadTroops()
	if len(troops) < 3 {
		fmt.Println("❌ Not enough troops.")
		for _, conn := range conns {
			if conn != nil {
				safeWriteToConn(conn, []byte("Server error: Not enough troops.\n"), 0)
			}
		}
		return
	}

	playerTroops := [2][]map[string]interface{}{
		PickRandomTroops(troops),
		PickRandomTroops(troops),
	}

	towers := []map[string]int{
		{"G1": 1000, "G2": 1000, "King": 2000},
		{"G1": 1000, "G2": 1000, "King": 2000},
	}

	current := 0

	// Send initial game start message to both players
	startMsg := "🎮 Game Started! Prepare for battle!\n"
	safeBroadcast(conns, []byte(startMsg))

	// Wait a moment for clients to process
	time.Sleep(100 * time.Millisecond)

	for i := range conns {
		if conns[i] != nil {
			playerMsg := fmt.Sprintf("You are Player %d\n", i+1)
			safeWriteToConn(conns[i], []byte(playerMsg), i+1)
		}
	}

	gameActive := true
	for gameActive {
		enemy := 1 - current
		fmt.Printf("🔄 Turn: Player %d\n", current+1)

		// Check if connections are still valid
		if conns[current] == nil || conns[enemy] == nil {
			fmt.Println("❌ Connection lost, ending game")
			break
		}

		// Notify both players whose turn it is
		turnMsg := fmt.Sprintf("🔄 Player %d's turn\n", current+1)
		safeBroadcast(conns, []byte(turnMsg))

		// Send troop info to current player only
		msg := "Your troops:\n"
		for i, t := range playerTroops[current] {
			name, _ := t["name"].(string)
			msg += fmt.Sprintf("%d. %s (ATK:%v DEF:%v)\n", i+1, name, t["atk"], t["def"])
		}
		msg += "Choose troop (1-3):\n"

		err := safeWriteToConn(conns[current], []byte(msg), current+1)
		if err != nil {
			fmt.Printf("❌ Cannot send troop info to player %d, ending game\n", current+1)
			break
		}

		// Send waiting message to other player
		waitMsg := "⏳ Waiting for opponent's move...\n"
		safeWriteToConn(conns[enemy], []byte(waitMsg), enemy+1)

		// Receive input with timeout
		conns[current].SetReadDeadline(time.Now().Add(30 * time.Second))
		reader := bufio.NewReader(conns[current])
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("⚠ Player %d input error: %v\n", current+1, err)
			timeoutMsg := "Timeout or error reading input. Skipping turn...\n"
			safeWriteToConn(conns[current], []byte(timeoutMsg), current+1)

			// Notify other player
			skipMsg := fmt.Sprintf("Player %d's turn was skipped due to timeout\n", current+1)
			safeWriteToConn(conns[enemy], []byte(skipMsg), enemy+1)

			current = enemy
			continue
		}

		choice, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || choice < 1 || choice > 3 {
			invalidMsg := "Invalid choice. Try again with 1, 2, or 3.\n"
			safeWriteToConn(conns[current], []byte(invalidMsg), current+1)
			continue // Don't switch turns, let them try again
		}
		choice--

		if choice >= len(playerTroops[current]) {
			invalidMsg := "Invalid troop index. Try again.\n"
			safeWriteToConn(conns[current], []byte(invalidMsg), current+1)
			continue
		}

		selected := playerTroops[current][choice]
		name, okName := selected["name"].(string)
		atkVal, okAtk := selected["atk"].(float64)
		defVal, okDef := selected["def"].(float64)
		if !okName || !okAtk || !okDef {
			invalidMsg := "Invalid troop data. Try again.\n"
			safeWriteToConn(conns[current], []byte(invalidMsg), current+1)
			continue
		}

		atk := int(atkVal)
		_ = int(defVal)

		// Target logic
		target := ""
		towerDef := 0
		if towers[enemy]["G1"] > 0 {
			target = "G1"
			towerDef = 100
		} else if towers[enemy]["G2"] > 0 {
			target = "G2"
			towerDef = 100
		} else {
			target = "King"
			towerDef = 300
		}

		// Damage calculation
		dmg := atk - towerDef
		if dmg < 0 {
			dmg = 0
		}
		towers[enemy][target] -= dmg
		if towers[enemy][target] < 0 {
			towers[enemy][target] = 0
		}

		// Send result to both players with more detailed info
		res := fmt.Sprintf("⚔️ Player %d's %s attacked Player %d's %s for %d damage. %s HP: %d\n",
			current+1, name, enemy+1, target, dmg, target, towers[enemy][target])

		safeBroadcast(conns, []byte(res))

		// Send tower status to both players
		statusMsg := fmt.Sprintf("🏰 Tower Status - Player 1: G1=%d G2=%d King=%d | Player 2: G1=%d G2=%d King=%d\n",
			towers[0]["G1"], towers[0]["G2"], towers[0]["King"],
			towers[1]["G1"], towers[1]["G2"], towers[1]["King"])

		safeBroadcast(conns, []byte(statusMsg))

		// Check win condition
		if towers[enemy]["King"] <= 0 {
			winMsg := fmt.Sprintf("🏆 Player %d wins! Game Over!\n", current+1)
			safeBroadcast(conns, []byte(winMsg))

			fmt.Printf("🎉 Game ended: Player %d wins!\n", current+1)

			// Give clients time to read the message
			time.Sleep(2 * time.Second)
			gameActive = false
			break
		}

		// Switch turn
		current = enemy

		// Small delay between turns
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("🎮 Game finished, cleaning up...")
}
