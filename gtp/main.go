package main

import (
	"fmt"
	"log"

	"github.com/rooklift/sgf"
)

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

func main() {
	filePath := "example.sgf"
	node, err := sgf.Load(filePath)
	if err != nil {
		log.Fatalf("Error loading SGF file: %v", err)
	}

	// Traverse the SGF file and convert positions to GTP coordinates
	traverseAndConvert(node)
}

// traverseAndConvert traverses the SGF nodes and converts the positions to GTP coordinates
func traverseAndConvert(node *sgf.Node) {
	for _, key := range []string{"B", "W"} {
		if move, ok := node.GetValue(key); ok {
			gtpMove := convertToGTP(move)
			fmt.Printf("Player: %s, Move: %s, GTP: %s\n", key, move, gtpMove)
		}
	}

	for _, child := range node.Children() {
		traverseAndConvert(child)
	}
}
