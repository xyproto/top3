// main.go

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// KataGoQuery represents a query to be sent to KataGo
type KataGoQuery struct {
	ID            string      `json:"id"`
	InitialStones [][2]string `json:"initialStones"`
	Moves         [][2]string `json:"moves"`
	Rules         string      `json:"rules"`
	Komi          float64     `json:"komi"`
	BoardXSize    int         `json:"boardXSize"`
	BoardYSize    int         `json:"boardYSize"`
	AnalyzeTurns  []int       `json:"analyzeTurns"`
	MaxVisits     int         `json:"maxVisits"`
}

// KataGoResponse represents a response from KataGo
type KataGoResponse struct {
	ID         string `json:"id"`
	Error      string `json:"error,omitempty"`
	TurnNumber int    `json:"turnNumber,omitempty"`
	MoveInfos  []struct {
		Move    string  `json:"move"`
		Winrate float64 `json:"winrate"`
		Visits  int     `json:"visits"`
	} `json:"moveInfos,omitempty"`
}

func main() {
	filePath := "example.sgf"
	sgfContent, err := LoadSGF(filePath)
	if err != nil {
		log.Fatalf("Error loading SGF file: %v", err)
	}

	initialStones, moves, rules, komi, boardSize, err := parseSGF(sgfContent)
	if err != nil {
		log.Fatalf("Error parsing SGF file: %v", err)
	}

	query := KataGoQuery{
		ID:            "query1",
		InitialStones: initialStones,
		Moves:         moves,
		Rules:         rules,
		Komi:          komi,
		BoardXSize:    boardSize,
		BoardYSize:    boardSize,
		AnalyzeTurns:  []int{len(moves)},
		MaxVisits:     1000,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		log.Fatalf("Failed to marshal query to JSON: %v", err)
	}

	// Start KataGo Analysis engine
	cmd := exec.Command("katago", "analysis", "-config", "./analyze.cfg", "-model", "./g170-b30c320x2-s4824661760-d1229536699.bin.gz")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to get stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to get stderr: %v", err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start KataGo: %v", err)
	}

	// Read stderr to debug any issues
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("KataGo stderr: %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading stderr: %v\n", err)
		}
	}()

	// Send the query to KataGo
	fmt.Printf("Sending query: %s\n", string(queryJSON))
	if _, err := fmt.Fprintln(stdin, string(queryJSON)); err != nil {
		log.Fatalf("Failed to send query: %v", err)
	}

	// Process results
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Response: %s\n", line)

		var response KataGoResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			log.Fatalf("Failed to parse JSON response: %v", err)
		}

		if response.Error != "" {
			log.Fatalf("KataGo error: %s", response.Error)
		}

		fmt.Printf("Response ID: %s, Turn Number: %d\n", response.ID, response.TurnNumber)
		for _, moveInfo := range response.MoveInfos {
			fmt.Printf("Move: %s, Winrate: %.2f, Visits: %d\n", moveInfo.Move, moveInfo.Winrate, moveInfo.Visits)
		}
		break // Stop after receiving the first response
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading stdout: %v\n", err)
	}

	// Close KataGo process
	if err := cmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to kill KataGo process: %v", err)
	}
}
