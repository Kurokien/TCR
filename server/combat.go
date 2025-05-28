package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Load troop data from JSON file
func LoadTroops() []map[string]interface{} {
	file, err := os.ReadFile("data/troops.json")
	if err != nil {
		fmt.Println("Error reading troops.json:", err)
		os.Exit(1)
	}
	var troops []map[string]interface{}
	json.Unmarshal(file, &troops)
	return troops
}

// Randomly pick 3 troops from the list
func PickRandomTroops(troops []map[string]interface{}) []map[string]interface{} {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(troops), func(i, j int) {
		troops[i], troops[j] = troops[j], troops[i]
	})
	return troops[:3]
}
