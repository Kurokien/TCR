package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func StartGame(players []*Player, conns []net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("‼ Panic in StartGame:", r)
		}
	}()

	troops := LoadTroops()
	if len(troops) < 3 {
		fmt.Println("❌ Not enough troops.")
		for _, conn := range conns {
			conn.Write([]byte("Server error: Not enough troops.\n"))
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
	turn := 1

	for {
		enemy := 1 - current
		fmt.Printf("🔁 Turn %d: Player %d\n", turn, current+1)

		msg := "Your troops:\n"
		for i, t := range playerTroops[current] {
			name, _ := t["name"].(string)
			msg += fmt.Sprintf("%d. %s (ATK:%v DEF:%v)\n", i+1, name, t["atk"], t["def"])
		}
		msg += "Choose troop (1-3):\n"
		_, err := conns[current].Write([]byte(msg))
		if err != nil {
			fmt.Printf("❌ Cannot send troop info to player %d, ending game\n", current+1)
			return
		}

		reader := bufio.NewReader(conns[current])
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading from player %d: %v\n", current+1, err)
			return
		}
		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || choice < 1 || choice > 3 {
			conns[current].Write([]byte("Invalid choice. Skipping turn.\n"))
			current = enemy
			turn++
			continue
		}

		choice--
		selected := playerTroops[current][choice]
		name, okName := selected["name"].(string)
		atkVal, okAtk := selected["atk"].(float64)
		if !okName || !okAtk {
			conns[current].Write([]byte("Invalid troop data. Skipping turn.\n"))
			current = enemy
			turn++
			continue
		}
		atk := int(atkVal)

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

		dmg := atk - towerDef
		if dmg < 0 {
			dmg = 0
		}
		towers[enemy][target] -= dmg
		if towers[enemy][target] < 0 {
			towers[enemy][target] = 0
		}

		res := fmt.Sprintf("%s attacked %s for %d damage. Remaining HP: %d\n", name, target, dmg, towers[enemy][target])
		for i := range conns {
			if _, err := conns[i].Write([]byte(res)); err != nil {
				fmt.Printf("❌ Error sending result to player %d: %v\n", i+1, err)
				return
			}
		}

		if towers[enemy]["King"] <= 0 {
			msg := fmt.Sprintf("🏆 Player %d wins!\n", current+1)
			for i := range conns {
				if _, err := conns[i].Write([]byte(msg)); err != nil {
					fmt.Printf("❌ Error sending win message to player %d: %v\n", i+1, err)
				}
				conns[i].Close()
			}
			break
		}

		current = enemy
		turn++
	}
}
