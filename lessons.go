package main

type lesson struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	FEN   string `json:"-"`
	Move  string `json:"-"`
	Why   string `json:"why"`
}

var lessons = []lesson{
	{"The rook", "Rooks travel straight along a rank or file. Tap the rook on d4, then move it to a4.", "7k/8/8/8/3R4/8/8/7K w - - 0 1", "d4a4", "Exactly. A rook can travel any number of clear squares in a straight line."},
	{"The bishop", "Bishops move diagonally and stay on the same square colour. Move the bishop from d4 to g7.", "7k/p7/8/8/3B4/8/8/7K w - - 0 1", "d4g7", "Good. The bishop followed the diagonal and now gives check."},
	{"The knight", "A knight moves two squares one way and one square sideways. It can jump over other pieces. Move d4 to f5.", "7k/p7/8/8/3N4/8/8/7K w - - 0 1", "d4f5", "That is the knight's L-shaped move. Legal destinations are marked with dots."},
	{"The pawn", "Pawns move forward and capture diagonally. From its starting square, a pawn may advance two squares if both are clear. Play e2 to e4.", "7k/8/8/8/8/8/4P3/7K w - - 0 1", "e2e4", "Well done. Later, this pawn can only move one square forward at a time."},
	{"Keep your king safe", "Castling moves your king and a rook together. Neither may have moved, the path must be clear, and the king cannot cross an attacked square. Move e1 to g1.", "4k3/8/8/8/8/8/8/R3K2R w KQ - 0 1", "e1g1", "Castled. The rook moved to f1 automatically. You cannot castle out of check."},
	{"Your first checkmate", "Check means the king is attacked. Checkmate means it has no legal escape. Move the queen from f7 to g7.", "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1", "f7g7", "Checkmate! Your king protects the queen, and the black king has nowhere safe to go."},
	{"Pawn promotion", "A pawn that reaches the far rank becomes a queen, rook, bishop or knight. Move a7 to a8 and choose a queen.", "7k/P7/8/8/8/8/8/7K w - - 0 1", "a7a8q", "Promoted. A new queen is usually the strongest choice, but you may choose another piece."},
	{"En passant", "Black just moved d7 to d5. Your pawn on e5 may capture it as if it moved only one square. Play e5 to d6 now.", "7k/8/8/3pP3/8/8/8/7K w - d6 0 1", "e5d6", "En passant! This special capture is available only immediately after the two-square pawn move."},
	{"Look for free material", "Before every move: check for checks, captures and threats. Here the black queen is unprotected. Capture it with the rook from d1.", "7k/8/8/3q4/8/8/8/3R2K1 w - - 0 1", "d1d5", "Excellent. Taking a free queen is worth far more than making a quiet developing move."},
}
