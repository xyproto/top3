package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/rooklift/sgf"
	"gopkg.in/yaml.v2"
)

// MoveInfo represents information about a move
type MoveInfo struct {
	Player  string
	Move    string
	Winrate float64
	Drop    float64
}

// AnalysisRequest represents the request structure for KataGo
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

// AnalysisResponse represents the response structure from KataGo
type AnalysisResponse struct {
	ID        string        `json:"id"`
	MoveInfos []MoveInfoExt `json:"moveInfos"`
}

// MoveInfoExt extends MoveInfo with extra information
type MoveInfoExt struct {
	Move    string  `json:"move"`
	Winrate float64 `json:"winrate"`
}

// Options represents configuration options
type Options struct {
	KataGo struct {
		Path      string `yaml:"path"`
		Arguments string `yaml:"arguments"`
		Model     string `yaml:"model"`
		Config    string `yaml:"config"`
	} `yaml:"katago"`
	Analysis struct {
		Rules      string  `yaml:"rules"`
		Komi       float64 `yaml:"komi"`
		BoardXSize int     `yaml:"boardXSize"`
		BoardYSize int     `yaml:"boardYSize"`
		MaxVisits  int     `yaml:"maxVisits"`
	} `yaml:"analysis"`
	SGF struct {
		MaxWinRateDropForGoodMove   float64 `yaml:"maxWinrateDropForGoodMove"`
		MinWinRateDropForBadMove    float64 `yaml:"minWinrateDropForBadMove"`
		MinWinRateDropForBadHotSpot float64 `yaml:"minWinrateDropForBadHotSpot"`
		ShowVariationsAfterLastMove bool    `yaml:"showVariationsAfterLastMove"`
		MinWinRateDropForVariations float64 `yaml:"minWinrateDropForVariations"`
		ShowBadVariations           bool    `yaml:"showBadVariations"`
		MaxVariationsForEachMove    int     `yaml:"maxVariationsForEachMove"`
		FileSuffix                  string  `yaml:"fileSuffix"`
	} `yaml:"sgf"`
}

func main() {
	// Define command-line flags
	var analysisOpts string
	var sgfOpts string
	var katagoOpts string
	var revisit int
	var saveJSON bool
	var analyzeJSON bool
	var help bool

	flag.StringVar(&analysisOpts, "a", "", "Options for KataGo Parallel Analysis Engine query")
	flag.StringVar(&sgfOpts, "g", "", "Options for making reviewed SGF files")
	flag.StringVar(&katagoOpts, "k", "", "Options for path and arguments of KataGo")
	flag.IntVar(&revisit, "r", 0, "For variation cases, Analyze again with maxVisits N")
	flag.BoolVar(&saveJSON, "s", false, "Save KataGo analysis as JSON files")
	flag.BoolVar(&analyzeJSON, "f", false, "Analyze by KataGo JSON files")
	flag.BoolVar(&help, "h", false, "Display this help and exit")

	flag.Parse()

	if help {
		displayHelp()
		return
	}

	if len(flag.Args()) == 0 {
		fmt.Println("Please specify SGF/GIB files.")
		displayHelp()
		return
	}

	filePaths := flag.Args()

	// Load configuration
	opts, err := loadConfig("analyze-sgf.yml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Set default values for KataGo model and config if not specified
	if opts.KataGo.Model == "" {
		opts.KataGo.Model = "model.bin.gz"
	}
	if opts.KataGo.Config == "" {
		opts.KataGo.Config = "analyze.cfg"
	}

	// Override options if provided through command-line
	if analysisOpts != "" {
		analysisOverrides := parseOptions(analysisOpts)
		for k, v := range analysisOverrides {
			switch k {
			case "rules":
				opts.Analysis.Rules = v
			case "komi":
				opts.Analysis.Komi = parseFloat(v)
			case "boardXSize":
				opts.Analysis.BoardXSize = parseInt(v)
			case "boardYSize":
				opts.Analysis.BoardYSize = parseInt(v)
			case "maxVisits":
				opts.Analysis.MaxVisits = parseInt(v)
			}
		}
	}

	if sgfOpts != "" {
		sgfOverrides := parseOptions(sgfOpts)
		for k, v := range sgfOverrides {
			switch k {
			case "maxWinrateDropForGoodMove":
				opts.SGF.MaxWinRateDropForGoodMove = parseFloat(v)
			case "minWinrateDropForBadMove":
				opts.SGF.MinWinRateDropForBadMove = parseFloat(v)
			case "minWinrateDropForBadHotSpot":
				opts.SGF.MinWinRateDropForBadHotSpot = parseFloat(v)
			case "showVariationsAfterLastMove":
				opts.SGF.ShowVariationsAfterLastMove = parseBool(v)
			case "minWinrateDropForVariations":
				opts.SGF.MinWinRateDropForVariations = parseFloat(v)
			case "showBadVariations":
				opts.SGF.ShowBadVariations = parseBool(v)
			case "maxVariationsForEachMove":
				opts.SGF.MaxVariationsForEachMove = parseInt(v)
			case "fileSuffix":
				opts.SGF.FileSuffix = v
			}
		}
	}

	if katagoOpts != "" {
		katagoOverrides := parseOptions(katagoOpts)
		for k, v := range katagoOverrides {
			switch k {
			case "path":
				opts.KataGo.Path = v
			case "arguments":
				opts.KataGo.Arguments = v
			case "model":
				opts.KataGo.Model = v
			case "config":
				opts.KataGo.Config = v
			}
		}
	}

	// Process each file
	for _, filePath := range filePaths {
		processFile(filePath, opts, revisit, saveJSON, analyzeJSON)
	}
}

func displayHelp() {
	fmt.Println(`Usage: analyze-sgf [-a=OPTS] [-g=OPTS] [-k=OPTS] [-s] [-f] FILE ...

Option:
  -a, --analysis=OPTS     Options for KataGo Parallel Analysis Engine query
  -g, --sgf=OPTS          Options for making reviewed SGF files
  -k, --katago=OPTS       Options for path and arguments of KataGo
  -r, --revisit=N         For variation cases, Analyze again with maxVisits N
  -s                      Save KataGo analysis as JSON files
  -f                      Analyze by KataGo JSON files
  -h, --help              Display this help and exit

Examples:
  analyze-sgf baduk-1.sgf baduk-2.gib
  analyze-sgf 'https://www.cyberoro.com/gibo_new/giboviewer/......'
  analyze-sgf -a 'maxVisits:16400,analyzeTurns:[197,198]' baduk.sgf
  analyze-sgf -f baduk.json
  analyze-sgf -g 'maxVariationsForEachMove:15' -r 20000 baduk.sgf`)
}

func parseOptions(opts string) map[string]string {
	options := make(map[string]string)
	for _, opt := range strings.Split(opts, ",") {
		parts := strings.SplitN(opt, ":", 2)
		if len(parts) == 2 {
			options[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return options
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}

func processFile(filePath string, opts Options, revisit int, saveJSON bool, analyzeJSON bool) {
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
	go kataGoAnalyzer(opts, requestCh, responseCh, &wg)

	moveEvaluations := make([]MoveInfo, 0)

	for i := range moves {
		request := AnalysisRequest{
			ID:            fmt.Sprintf("analysis_%d", i),
			InitialStones: initialStones,
			Moves:         moves[:i+1],
			Rules:         opts.Analysis.Rules,
			Komi:          opts.Analysis.Komi,
			BoardXSize:    opts.Analysis.BoardXSize,
			BoardYSize:    opts.Analysis.BoardYSize,
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

	// Save JSON if required
	if saveJSON {
		saveAnalysisAsJSON(filePath, initialStones, moves, moveEvaluations)
	}
}

func saveAnalysisAsJSON(filePath string, initialStones [][2]string, moves [][2]string, moveEvaluations []MoveInfo) {
	jsonData := map[string]interface{}{
		"initialStones": initialStones,
		"moves":         moves,
		"evaluations":   moveEvaluations,
	}

	file, err := os.Create(strings.TrimSuffix(filePath, ".sgf") + ".json")
	if err != nil {
		log.Fatalf("Error creating JSON file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonData); err != nil {
		log.Fatalf("Error writing JSON file: %v", err)
	}

	fmt.Printf("generated: %s\n", file.Name())
}

// kataGoAnalyzer runs KataGo and handles requests for analysis
func kataGoAnalyzer(opts Options, requestCh <-chan AnalysisRequest, responseCh chan<- AnalysisResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	// Start KataGo in analysis mode
	args := strings.Fields(opts.KataGo.Arguments)
	cmd := exec.Command(opts.KataGo.Path, args...)
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

// loadConfig loads the configuration from a YAML file
func loadConfig(filename string) (Options, error) {
	var opts Options
	file, err := os.Open(filename)
	if err != nil {
		return opts, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&opts)
	return opts, err
}
