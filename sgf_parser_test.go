package main

import (
	"testing"
)

func TestConvertToGTP(t *testing.T) {
	testCases := []struct {
		sgfPos  string
		gtpPos  string
		isValid bool
	}{
		{"aa", "A19", true},
		{"sq", "S17", true},
		{"pd", "Q16", true},
		{"zz", "", false},
		{"b@", "", false},
	}

	for _, tc := range testCases {
		gtpPos, isValid := convertToGTP(tc.sgfPos)
		if isValid != tc.isValid || gtpPos != tc.gtpPos {
			t.Errorf("convertToGTP(%s) = (%s, %v), expected (%s, %v)", tc.sgfPos, gtpPos, isValid, tc.gtpPos, tc.isValid)
		}
	}
}

func TestExtractPositions(t *testing.T) {
	prop := "AB[aa][bb][cc]"
	expected := []string{"aa", "bb", "cc"}
	positions := extractPositions(prop)
	if len(positions) != len(expected) {
		t.Fatalf("expected %d positions, got %d", len(expected), len(positions))
	}
	for i, pos := range positions {
		if pos != expected[i] {
			t.Errorf("expected position %d to be %s, got %s", i, expected[i], pos)
		}
	}
}

func TestParseSGF(t *testing.T) {
	sgfContent := "(;B[pd];W[dp];B[qp];W[dc];B[oq];W[qj];B[ce];W[dh];B[cc];W[cb];B[cd];W[bb];B[fe];W[fc];B[ed];W[gc];B[cq];W[dq];B[cp];W[co];B[dr];W[er];B[cr];W[fq];B[bo];W[cn];B[bn];W[cm];B[qf];W[qm];B[jp];W[ef];B[ec];W[eb];B[dd];W[db];B[bc];W[bm];B[ar];W[fh];B[bg];W[bh];B[cg];W[ch];B[he];W[id];B[ie];W[jd];B[lc];W[hh];B[im];W[gm];B[ik];W[gk];B[ho];W[an];B[ap];W[nd];B[nc];W[oc];B[pc];W[ob];B[pb];W[od];B[le];W[mc];B[md];W[mb];B[ne];W[oe];B[je];W[kd];B[hd];W[ld];B[hc];W[ke];B[hb];W[ab];B[gb];W[fb];B[fo];W[en];B[eo];W[do];B[fn];W[fm];B[pl];W[ql];B[pm];W[pk];B[nl];W[pn];B[on];W[oo];B[po];W[qn];B[nn];W[rp];B[rq];W[ro];B[op];W[ir];B[jr];W[iq];B[jq];W[hn];B[in];W[go];B[gn];W[hm];B[gp];W[ip];B[io];W[qo];B[pp];W[sq])"

	_, moves, _, _, _, err := parseSGF(sgfContent)
	if err != nil {
		t.Fatalf("parseSGF failed: %v", err)
	}

	expectedMoves := [][2]string{
		{"B", "Q16"}, {"W", "D4"}, {"B", "R4"}, {"W", "D17"}, {"B", "P3"}, {"W", "R10"},
		{"B", "C15"}, {"W", "D12"}, {"B", "C17"}, {"W", "C18"}, {"B", "C16"}, {"W", "B18"},
		{"B", "F15"}, {"W", "F17"}, {"B", "E16"}, {"W", "G17"}, {"B", "C3"}, {"W", "D3"},
		{"B", "C4"}, {"W", "C5"}, {"B", "D2"}, {"W", "E2"}, {"B", "C2"}, {"W", "F3"},
		{"B", "B5"}, {"W", "C6"}, {"B", "B6"}, {"W", "C7"}, {"B", "R14"}, {"W", "R7"},
		{"B", "K4"}, {"W", "E14"}, {"B", "E17"}, {"W", "E18"}, {"B", "D16"}, {"W", "D18"},
		{"B", "B17"}, {"W", "B7"}, {"B", "A2"}, {"W", "F12"}, {"B", "B13"}, {"W", "B12"},
		{"B", "C13"}, {"W", "C12"}, {"B", "H15"}, {"W", "J16"}, {"B", "J15"}, {"W", "K16"},
		{"B", "M17"}, {"W", "H12"}, {"B", "J7"}, {"W", "G7"}, {"B", "J9"}, {"W", "G9"},
		{"B", "H5"}, {"W", "A6"}, {"B", "A4"}, {"W", "O16"}, {"B", "O17"}, {"W", "P17"},
		{"B", "Q17"}, {"W", "P18"}, {"B", "Q18"}, {"W", "P16"}, {"B", "M15"}, {"W", "N17"},
		{"B", "N16"}, {"W", "N18"}, {"B", "O15"}, {"W", "P15"}, {"B", "K15"}, {"W", "L16"},
		{"B", "H16"}, {"W", "M16"}, {"B", "H17"}, {"W", "L15"}, {"B", "H18"}, {"W", "A18"},
		{"B", "G18"}, {"W", "F18"}, {"B", "F5"}, {"W", "E6"}, {"B", "E5"}, {"W", "D5"},
		{"B", "F6"}, {"W", "F7"}, {"B", "Q8"}, {"W", "R8"}, {"B", "Q7"}, {"W", "Q9"},
		{"B", "O8"}, {"W", "Q6"}, {"B", "P6"}, {"W", "P5"}, {"B", "Q5"}, {"W", "R6"},
		{"B", "O6"}, {"W", "S4"}, {"B", "S3"}, {"W", "S5"}, {"B", "P4"}, {"W", "J2"},
		{"B", "K2"}, {"W", "J3"}, {"B", "K3"}, {"W", "H6"}, {"B", "J6"}, {"W", "G5"},
		{"B", "G6"}, {"W", "H7"}, {"B", "G4"}, {"W", "J4"}, {"B", "J5"}, {"W", "R5"},
		{"B", "Q4"}, {"W", "S17"},
	}

	if len(moves) != len(expectedMoves) {
		t.Fatalf("expected %d moves, got %d", len(expectedMoves), len(moves))
	}

	for i, move := range moves {
		if move != expectedMoves[i] {
			t.Errorf("expected move %d to be %v, got %v", i, expectedMoves[i], move)
		}
	}
}
