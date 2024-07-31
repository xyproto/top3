package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rooklift/sgf"
)

// MoveInfo represents information about a move
type MoveInfo struct {
	Player  string
	Move    string
	Winrate float64
	Drop    float64
}

func main() {
	filePath := "example.sgf"
	node, err := LoadSGF(filePath)
	if err != nil {
		log.Fatalf("Error loading SGF file: %v", err)
	}

	initialStones, moves := extractMoves(node)

	// Analyze each move and record its evaluation
	moveEvaluations := analyzeMoves(initialStones, moves)

	// Find the worst move for black
	worstBlackMove := findWorstMove(moveEvaluations, "B")

	// Output the worst move for black
	fmt.Printf("Worst move for Black: %s with winrate drop %.2f\n", worstBlackMove.Move, worstBlackMove.Drop)
}

// LoadSGF loads the SGF file and returns the root node
func LoadSGF(filePath string) (*sgf.Node, error) {
	node, err := sgf.Load(filePath)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// extractMoves extracts initial stones and moves from the SGF file
func extractMoves(node *sgf.Node) (initialStones [][2]string, moves [][2]string) {
	initialStones = make([][2]string, 0)
	moves = make([][2]string, 0)

	for _, key := range []string{"AB", "AW"} {
		for _, value := range node.AllValues(key) {
			player := "B"
			if key == "AW" {
				player = "W"
			}
			initialStones = append(initialStones, [2]string{player, convertToGTP(value)})
		}
	}

	for _, child := range node.Children() {
		for _, key := range []string{"B", "W"} {
			if move, ok := child.GetValue(key); ok {
				moves = append(moves, [2]string{key, convertToGTP(move)})
			}
		}
	}

	return initialStones, moves
}

// convertToGTP converts an SGF coordinate to a GTP coordinate
func convertToGTP(sgfCoord string) string {
	if sgfCoord == "" {
		return "pass"
	}

	colChar := sgfCoord[0]
	rowChar := sgfCoord[1]

	col := colChar - 'a'
	row := 19 - (rowChar - 'a')

	// Skip the 'I' column in GTP format
	if col >= 8 {
		col++
	}

	gtpCoord := fmt.Sprintf("%c%d", col+'A', row)
	return gtpCoord
}

// analyzeMoves analyzes each move and records its evaluation using GTP commands
func analyzeMoves(initialStones [][2]string, moves [][2]string) []MoveInfo {
	moveEvaluations := make([]MoveInfo, 0)

	// Start KataGo in GTP mode
	cmd := exec.Command("katago", "gtp", "-config", "gtp_example.cfg", "-model", "model.bin.gz")
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

	reader := bufio.NewReader(stdout)
	sendGTPCommand(stdin, "boardsize 19", reader)
	sendGTPCommand(stdin, "clear_board", reader)

	for _, stone := range initialStones {
		sendGTPCommand(stdin, fmt.Sprintf("play %s %s", stone[0], stone[1]), reader)
	}

	//var lastWinrate float64

	for i, move := range moves {
		// Request analysis before the move
		responseBefore := sendGTPCommand(stdin, fmt.Sprintf("kata-genmove_analyze %s", move[0]), reader)
		winrateBefore := extractWinrate(responseBefore)

		sendGTPCommand(stdin, fmt.Sprintf("play %s %s", move[0], move[1]), reader)

		// Request analysis after the move
		responseAfter := sendGTPCommand(stdin, fmt.Sprintf("kata-genmove_analyze %s", move[0]), reader)
		winrateAfter := extractWinrate(responseAfter)

		// Calculate the drop in winrate
		winrateDrop := winrateBefore - winrateAfter

		// Log the raw responses
		fmt.Printf("Raw response before move %d: %s\n", i+1, responseBefore)
		fmt.Printf("Raw response after move %d: %s\n", i+1, responseAfter)

		// Record the move evaluation
		moveEvaluations = append(moveEvaluations, MoveInfo{
			Player:  move[0],
			Move:    move[1],
			Winrate: winrateAfter,
			Drop:    winrateDrop,
		})

		//lastWinrate = winrateAfter

		// Wait a bit before starting the next query
		time.Sleep(1 * time.Second)
	}

	// Close KataGo process
	if err := cmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to kill KataGo process: %v", err)
	}

	return moveEvaluations
}

// sendGTPCommand sends a GTP command to KataGo and reads the response
func sendGTPCommand(stdin io.Writer, command string, reader *bufio.Reader) string {
	fmt.Printf("Sending command: %s\n", command)
	if _, err := fmt.Fprintln(stdin, command); err != nil {
		log.Fatalf("Failed to send command: %v", err)
	}

	var response bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}
		if line == "\n" {
			break
		}
		response.WriteString(line)
	}
	fmt.Printf("Response: %s\n", response.String())
	return response.String()
}

// extractWinrate extracts the winrate from the GTP response
func extractWinrate(response string) float64 {
	lines := strings.Split(response, "\n")
	winrateRegex := regexp.MustCompile(`info move .* winrate ([0-9.]+)`)

	for _, line := range lines {
		if strings.HasPrefix(line, "info move") {
			matches := winrateRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				winrate, err := strconv.ParseFloat(matches[1], 64)
				if err != nil {
					log.Printf("Failed to parse winrate: %v\n", err)
					continue
				}
				return winrate
			}
		}
	}

	return 0.0
}

// findWorstMove finds the worst move for a given player
func findWorstMove(moveEvaluations []MoveInfo, player string) MoveInfo {
	var worstMove MoveInfo
	worstDrop := 0.0

	for _, moveInfo := range moveEvaluations {
		if moveInfo.Player == player && moveInfo.Drop > worstDrop {
			worstMove = moveInfo
			worstDrop = moveInfo.Drop
		}
	}

	return worstMove
}
