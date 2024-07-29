// sgf_parser.go

package main

import (
	"fmt"
	"os"
	"regexp"
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
	re := regexp.MustCompile(`;([^;()]+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	initialStones = make([][2]string, 0)
	moves = make([][2]string, 0)
	rules = "tromp-taylor"
	komi = 7.5
	boardSize = 19

	for _, match := range matches {
		props := match[1]
		if strings.HasPrefix(props, "B[") || strings.HasPrefix(props, "W[") {
			player := string(props[0])
			pos := props[2:4]
			gtpPos, valid := convertToGTP(pos)
			if !valid {
				return nil, nil, "", 0, 0, fmt.Errorf("invalid SGF position: %s", pos)
			}
			moves = append(moves, [2]string{player, gtpPos})
		} else {
			handleSGFProperty(props, &initialStones, &rules, &komi, &boardSize)
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
			gtpPos, valid := convertToGTP(pos)
			if valid {
				*initialStones = append(*initialStones, [2]string{"B", gtpPos})
			}
		}
	case strings.HasPrefix(props, "AW["):
		positions := extractPositions(props)
		for _, pos := range positions {
			gtpPos, valid := convertToGTP(pos)
			if valid {
				*initialStones = append(*initialStones, [2]string{"W", gtpPos})
			}
		}
	}
}

// extractPositions extracts positions from an SGF property
func extractPositions(prop string) []string {
	re := regexp.MustCompile(`\[([a-z]{2})\]`)
	matches := re.FindAllStringSubmatch(prop, -1)

	positions := make([]string, len(matches))
	for i, match := range matches {
		positions[i] = match[1]
	}

	return positions
}

// convertToGTP converts an SGF position to GTP format
func convertToGTP(sgfPos string) (string, bool) {
	if len(sgfPos) != 2 {
		return "", false
	}
	col := sgfPos[0] - 'a'
	if col >= 8 { // Skip the 'I' column
		col++
	}
	row := 19 - (sgfPos[1] - 'a')
	if col < 0 || col >= 19 || row < 1 || row > 19 {
		return "", false
	}
	return fmt.Sprintf("%c%d", col+'A', row), true
}
