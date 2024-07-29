// katago_test.go

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
)

// TestKataGoCommunication tests the communication with KataGo
func TestKataGoCommunication(t *testing.T) {
	query := KataGoQuery{
		ID:            "query1",
		InitialStones: [][2]string{{"B", "D4"}},
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
		t.Fatalf("Failed to marshal query to JSON: %v", err)
	}

	cmd := exec.Command("katago", "analysis", "-config", "./analyze.cfg", "-model", "/opt/homebrew/Cellar/katago/1.14.1/share/katago/g170-b30c320x2-s4824661760-d1229536699.bin.gz")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start KataGo: %v", err)
	}

	// Read stderr to debug any issues
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("KataGo stderr: %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			t.Logf("Error reading stderr: %v\n", err)
		}
	}()

	// Send the query to KataGo
	if _, err := fmt.Fprintln(stdin, string(queryJSON)); err != nil {
		t.Fatalf("Failed to send query: %v", err)
	}

	// Process results
	var responseBuffer bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		responseBuffer.WriteString(line)
		break // Stop after receiving the first response
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading stdout: %v", err)
	}

	var response KataGoResponse
	if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response.Error != "" {
		t.Fatalf("KataGo error: %s", response.Error)
	}

	t.Logf("Response ID: %s, Turn Number: %d\n", response.ID, response.TurnNumber)
	for _, moveInfo := range response.MoveInfos {
		t.Logf("Move: %s, Winrate: %.2f, Visits: %d\n", moveInfo.Move, moveInfo.Winrate, moveInfo.Visits)
	}

	// Close KataGo process
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill KataGo process: %v", err)
	}
}
