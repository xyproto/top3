package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sort"
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
	// Define the initial board position with two black stones and two white stones
	initialStones := [][2]string{
		{"B", "D4"}, {"B", "Q16"},
		{"W", "D16"}, {"W", "Q4"},
	}

	query := KataGoQuery{
		ID:            "query1",
		InitialStones: initialStones,
		Moves:         [][2]string{},
		Rules:         "tromp-taylor",
		Komi:          7.5,
		BoardXSize:    19,
		BoardYSize:    19,
		AnalyzeTurns:  []int{0},
		MaxVisits:     1000,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		log.Fatalf("Failed to marshal query to JSON: %v", err)
	}

	// Start KataGo Analysis engine
	cmd := exec.Command("katago", "analysis", "-config", "analyze.cfg", "-model", "model.bin.gz")
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
	stdin.Close()

	// Process results
	reader := bufio.NewReader(stdout)
	var responseStr bytes.Buffer

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading stdout: %v", err)
		}
		responseStr.Write(line)
		if !isPrefix {
			var response KataGoResponse
			if err := json.Unmarshal(responseStr.Bytes(), &response); err != nil {
				log.Printf("Failed to parse JSON response: %v\n", err)
				continue
			}
			if response.Error != "" {
				log.Fatalf("KataGo error: %s", response.Error)
			}
			fmt.Printf("Response ID: %s, Turn Number: %d\n", response.ID, response.TurnNumber)
			printTopMoves(response.MoveInfos)
			responseStr.Reset() // Clear the buffer for the next line
		}
	}

	// Close KataGo process
	if err := cmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to kill KataGo process: %v", err)
	}
}

func printTopMoves(moveInfos []struct {
	Move    string  `json:"move"`
	Winrate float64 `json:"winrate"`
	Visits  int     `json:"visits"`
}) {
	if len(moveInfos) == 0 {
		fmt.Println("No move information available.")
		return
	}

	// Sort moves by winrate
	sort.Slice(moveInfos, func(i, j int) bool {
		return moveInfos[i].Winrate > moveInfos[j].Winrate
	})

	fmt.Println("Top 3 moves:")
	for i := 0; i < 3 && i < len(moveInfos); i++ {
		moveInfo := moveInfos[i]
		fmt.Printf("Move: %s, Winrate: %.2f%%, Visits: %d\n", moveInfo.Move, moveInfo.Winrate*100, moveInfo.Visits)
	}
}
