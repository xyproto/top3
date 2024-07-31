package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"sync"

	"github.com/rooklift/sgf"
)

// MoveInfo represents information about a move
type MoveInfo struct {
	Player  string
	Move    string
	Winrate float64
	Drop    float64
}

type AnalysisRequest struct {
	ID            string      `json:"id"`
	InitialStones [][2]string `json:"initialStones,omitempty"`
	Moves         [][2]string `json:"moves"`
	Rules         string      `json:"rules"`
	Komi          float64     `json:"komi"`
	BoardXSize    int         `json:"boardXSize"`
	BoardYSize    int         `json:"boardYSize"`
	AnalyzeTurns  []int       `json:"analyzeTurns"`
}

type AnalysisResponse struct {
	ID        string        `json:"id"`
	MoveInfos []MoveInfoExt `json:"moveInfos"`
}

type MoveInfoExt struct {
	Move    string  `json:"move"`
	Winrate float64 `json:"winrate"`
}

func main() {
	filePath := "example.sgf"
	node, err := LoadSGF(filePath)
	if err != nil {
		log.Fatalf("Error loading SGF file: %v", err)
	}

	initialStones, moves := extractMoves(node)

	// Channel to send requests to the KataGo goroutine
	requestCh := make(chan AnalysisRequest)
	// Channel to receive responses from the KataGo goroutine
	responseCh := make(chan AnalysisResponse)

	var wg sync.WaitGroup

	// Start the KataGo goroutine
	wg.Add(1)
	go kataGoAnalyzer(requestCh, responseCh, &wg)

	moveEvaluations := make([]MoveInfo, 0)

	for i := range moves {
		request := AnalysisRequest{
			ID:            fmt.Sprintf("analysis_%d", i),
			InitialStones: initialStones,
			Moves:         moves[:i+1],
			Rules:         "tromp-taylor",
			Komi:          7.5,
			BoardXSize:    19,
			BoardYSize:    19,
			AnalyzeTurns:  []int{i},
		}

		// Send request to KataGo goroutine
		requestCh <- request

		// Wait for the response
		response := <-responseCh

		// Process the response
		if len(response.MoveInfos) > 0 {
			moveInfo := MoveInfo{
				Player:  moves[i][0],
				Move:    moves[i][1],
				Winrate: response.MoveInfos[0].Winrate,
				Drop:    0.5 - response.MoveInfos[0].Winrate, // Assuming initial winrate is 0.5
			}
			moveEvaluations = append(moveEvaluations, moveInfo)
		}
	}

	// Close the request channel to signal the KataGo goroutine to exit
	close(requestCh)
	// Wait for the KataGo goroutine to finish
	wg.Wait()

	// Find the worst moves
	worstMoves := findWorstMoves(moveEvaluations, 3)

	// Output the worst moves
	for i, move := range worstMoves {
		fmt.Printf("Worst move %d: %s by %s with winrate drop %.2f\n", i+1, move.Move, move.Player, move.Drop)
	}
}

// kataGoAnalyzer runs KataGo and handles requests for analysis
func kataGoAnalyzer(requestCh <-chan AnalysisRequest, responseCh chan<- AnalysisResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	// Start KataGo in analysis mode
	cmd := exec.Command("katago", "analysis", "-config", "analysis_example.cfg", "-model", "model.bin.gz")
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
	writer := bufio.NewWriter(stdin)

	for request := range requestCh {
		// Send analysis request to KataGo
		requestJSON, err := json.Marshal(request)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}
		fmt.Fprintf(writer, "%s\n", requestJSON)
		writer.Flush()

		// Read response from KataGo
		responseJSON, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}

		var response AnalysisResponse
		if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
			log.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Send response to the main goroutine
		responseCh <- response
	}

	// Close KataGo process
	if err := cmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to kill KataGo process: %v", err)
	}
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
			player := "black"
			if key == "AW" {
				player = "white"
			}
			initialStones = append(initialStones, [2]string{player, convertToGTP(value)})
		}
	}

	for _, child := range node.Children() {
		for _, key := range []string{"B", "W"} {
			if move, ok := child.GetValue(key); ok {
				player := "black"
				if key == "W" {
					player = "white"
				}
				moves = append(moves, [2]string{player, convertToGTP(move)})
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

// findWorstMoves finds the worst moves based on winrate drop
func findWorstMoves(moveEvaluations []MoveInfo, num int) []MoveInfo {
	// Sort moves by winrate drop in descending order
	sort.Slice(moveEvaluations, func(i, j int) bool {
		return moveEvaluations[i].Drop > moveEvaluations[j].Drop
	})

	if len(moveEvaluations) > num {
		return moveEvaluations[:num]
	}
	return moveEvaluations
}
