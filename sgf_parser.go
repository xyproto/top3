package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// LoadSGF loads and parses an SGF file from the given path
func LoadSGF(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read SGF file: %v", err)
	}

	return string(data), nil
}

// parseSGF parses the SGF content into initial stones and moves
func parseSGF(content string) (initialStones [][2]string, moves [][2]string, rules string, komi float64, boardSize int, err error) {
	content = strings.TrimSpace(content)
	if len(content) == 0 || content[0] != '(' || content[len(content)-1] != ')' {
		return nil, nil, "", 0, 0, fmt.Errorf("invalid SGF format")
	}

	content = content[1 : len(content)-1]
	nodes := strings.Split(content, ";")

	initialStones = make([][2]string, 0)
	moves = make([][2]string, 0)
	rules = "tromp-taylor"
	komi = 7.5
	boardSize = 19

	for _, node := range nodes {
		if len(node) == 0 {
			continue
		}
		if strings.HasPrefix(node, "B[") || strings.HasPrefix(node, "W[") {
			player := string(node[0])
			pos := strings.ToLower(node[2:4])
			log.Printf("Processing move: player=%s, pos=%s", player, pos)
			gtpPos, valid := convertToGTP(pos)
			if !valid {
				return nil, nil, "", 0, 0, fmt.Errorf("invalid SGF position: %s", pos)
			}
			log.Printf("Converted GTP position: %s", gtpPos)
			moves = append(moves, [2]string{player, gtpPos})
		} else {
			handleSGFProperty(node, &initialStones, &rules, &komi, &boardSize)
		}
	}

	return initialStones, moves, rules, komi, boardSize, nil
}

// handleSGFProperty handles various SGF properties and updates the relevant variables
func handleSGFProperty(props string, initialStones *[][2]string, rules *string, komi *float64, boardSize *int) {
	switch {
	case strings.HasPrefix(props, "SZ["):
		fmt.Sscanf(props, "SZ[%d]", boardSize)
	case strings.HasPrefix(props, "KM["):
		fmt.Sscanf(props, "KM[%f]", komi)
	case strings.HasPrefix(props, "RU["):
		*rules = strings.TrimSuffix(strings.TrimPrefix(props, "RU["), "]")
	case strings.HasPrefix(props, "AB["):
		positions := extractPositions(props)
		for _, pos := range positions {
			gtpPos, valid := convertToGTP(strings.ToLower(pos))
			if valid {
				*initialStones = append(*initialStones, [2]string{"B", gtpPos})
			}
		}
	case strings.HasPrefix(props, "AW["):
		positions := extractPositions(props)
		for _, pos := range positions {
			gtpPos, valid := convertToGTP(strings.ToLower(pos))
			if valid {
				*initialStones = append(*initialStones, [2]string{"W", gtpPos})
			}
		}
	}
}

// extractPositions extracts positions from an SGF property
func extractPositions(prop string) []string {
	positions := []string{}
	start := strings.Index(prop, "[")
	for start != -1 {
		end := strings.Index(prop[start:], "]")
		if end == -1 {
			break
		}
		end += start
		pos := prop[start+1 : end]
		positions = append(positions, pos)
		start = strings.Index(prop[end:], "[")
		if start == -1 {
			break
		}
		start += end
	}
	return positions
}

// convertToGTP converts an SGF position to GTP format
func convertToGTP(sgfPos string) (string, bool) {
	log.Printf("Converting SGF position: %s", sgfPos)

	if len(sgfPos) != 2 {
		log.Printf("Invalid SGF position length: %s", sgfPos)
		return "", false
	}

	colChar := sgfPos[0]
	rowChar := sgfPos[1]

	log.Printf("Column character: %c, Row character: %c", colChar, rowChar)

	col := colChar - 'a'
	row := 19 - (rowChar - 'a')

	log.Printf("Initial col=%d, row=%d", col, row)

	// Skip the 'I' column in GTP format
	if col >= 8 {
		col++
	}

	log.Printf("Adjusted col for GTP=%d", col)

	if col < 0 || col >= 19 || row < 1 || row > 19 {
		log.Printf("SGF position out of bounds: col=%d, row=%d", col, row)
		return "", false
	}

	gtpPos := fmt.Sprintf("%c%d", col+'A', row)
	log.Printf("Converted SGF position: %s to GTP position: %s", sgfPos, gtpPos)
	return gtpPos, true
}
