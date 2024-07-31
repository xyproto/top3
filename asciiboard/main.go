package main

import (
	"fmt"
	"log"

	"github.com/rooklift/sgf"
)

func main() {
	filePath := "example.sgf"
	node, err := sgf.Load(filePath)
	if err != nil {
		log.Fatalf("Error loading SGF file: %v", err)
	}

	printBoardAtMove(node, 20)
}

// printBoardAtMove prints the board position at the specified move number as ASCII graphics.
func printBoardAtMove(node *sgf.Node, moveNumber int) {
	currentNode := node
	moveCount := 0

	for currentNode != nil && moveCount < moveNumber {
		moveFound := false
		for _, child := range currentNode.Children() {
			for _, key := range []string{"B", "W"} {
				if _, ok := child.GetValue(key); ok {
					moveCount++
					if moveCount == moveNumber {
						printBoard(child.Board())
						return
					}
					moveFound = true
					currentNode = child
					break
				}
			}
			if moveFound {
				break
			}
		}
		if !moveFound {
			break
		}
	}
}

// printBoard prints the board in ASCII graphics.
func printBoard(board *sgf.Board) {
	if board == nil {
		fmt.Println("No board state available.")
		return
	}
	fmt.Print(board.String())
}
