package main

import (
	"github.com/notnil/chess"
	"math/rand"
	"sort"
	"time"
)

var pieceValues = map[chess.PieceType]int{chess.Pawn: 100, chess.Knight: 320, chess.Bishop: 335, chess.Rook: 500, chess.Queen: 900, chess.King: 20000}

type searcher struct {
	deadline time.Time
	nodes    int
	stopped  bool
}

func evaluate(pos *chess.Position) int {
	score := 0
	for square, piece := range pos.Board().SquareMap() {
		rank, file := int(square.Rank()), int(square.File())
		if piece.Color() == chess.Black {
			rank = 7 - rank
		}
		center := 7 - abs(2*file-7) - abs(2*rank-7)
		value := pieceValues[piece.Type()]
		switch piece.Type() {
		case chess.Pawn:
			value += rank*7 + center*2
		case chess.Knight:
			value += center * 9
			if rank == 0 {
				value -= 18
			}
		case chess.Bishop:
			value += center * 5
			if rank == 0 {
				value -= 12
			}
		case chess.Rook:
			if rank == 6 {
				value += 30
			}
		case chess.King:
			if rank == 0 && (file == 6 || file == 2) {
				value += 35
			}
			if rank > 1 {
				value -= rank * 8
			}
		}
		if piece.Color() == chess.White {
			score += value
		} else {
			score -= value
		}
	}
	if pos.Turn() == chess.Black {
		return -score
	}
	return score
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func ordered(pos *chess.Position) []*chess.Move {
	moves := pos.ValidMoves()
	rank := func(m *chess.Move) int {
		v := 0
		if m.HasTag(chess.Capture) {
			v = 10000 + 10*pieceValues[pos.Board().Piece(m.S2()).Type()] - pieceValues[pos.Board().Piece(m.S1()).Type()]
		}
		if m.Promo() != chess.NoPieceType {
			v += 9000 + pieceValues[m.Promo()]
		}
		if m.HasTag(chess.Check) {
			v += 100
		}
		return v
	}
	sort.SliceStable(moves, func(i, j int) bool { return rank(moves[i]) > rank(moves[j]) })
	return moves
}
func (s *searcher) negamax(pos *chess.Position, depth, alpha, beta, ply int) int {
	s.nodes++
	if s.nodes%128 == 0 && (time.Now().After(s.deadline) || s.nodes > 100000) {
		s.stopped = true
		return evaluate(pos)
	}
	moves := ordered(pos)
	if len(moves) == 0 {
		if pos.Status() == chess.Checkmate {
			return -100000 + ply
		}
		return 0
	}
	if depth == 0 {
		return evaluate(pos)
	}
	best := -1000000
	for _, m := range moves {
		v := -s.negamax(pos.Update(m), depth-1, -beta, -alpha, ply+1)
		if s.stopped {
			return v
		}
		if v > best {
			best = v
		}
		if v > alpha {
			alpha = v
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

// Search runs outside the GUI. It is intentionally modest: a learning opponent,
// rather than an advertised Elo or a Stockfish-strength engine.
func bestMove(pos *chess.Position, level int) *chess.Move {
	moves := ordered(pos)
	if len(moves) == 0 {
		return nil
	}
	budget := time.Duration(250+level*350) * time.Millisecond
	s := searcher{deadline: time.Now().Add(budget)}
	best := moves[0]
	maxDepth := level + 1
	if maxDepth > 4 {
		maxDepth = 4
	}
	for depth := 1; depth <= maxDepth; depth++ {
		iteration := best
		value := -1000000
		for _, m := range moves {
			v := -s.negamax(pos.Update(m), depth-1, -1000000, 1000000, 1)
			if s.stopped {
				break
			}
			if v > value {
				value = v
				iteration = m
			}
		}
		if s.stopped {
			break
		}
		best = iteration
	}
	if level == 1 && len(moves) > 2 {
		// Occasional small inaccuracies, never a random illegal move.
		if rand.Intn(5) == 0 {
			base := -evaluate(pos.Update(best))
			choices := []*chess.Move{}
			for _, m := range moves {
				if -evaluate(pos.Update(m)) >= base-55 {
					choices = append(choices, m)
				}
			}
			if len(choices) > 0 {
				best = choices[rand.Intn(len(choices))]
			}
		}
	}
	return best
}
