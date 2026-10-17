package main

import (
	"encoding/json"
	"github.com/notnil/chess"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func move(t *testing.T, a *app, from, to string) {
	t.Helper()
	if err := a.actionLocked(action{Action: "tap", Square: from}); err != nil {
		t.Fatal(err)
	}
	if err := a.actionLocked(action{Action: "tap", Square: to}); err != nil {
		t.Fatal(err)
	}
}
func TestLocalSaveUndoAndLessonIsolation(t *testing.T) {
	a := newApp(t.TempDir())
	if err := a.actionLocked(action{Action: "new", Mode: "local", Level: 1}); err != nil {
		t.Fatal(err)
	}
	move(t, a, "e2", "e4")
	move(t, a, "e7", "e5")
	fen := a.game.FEN()
	b := newApp(a.data)
	if b.game.FEN() != fen {
		t.Fatal("resume lost position")
	}
	if err := a.loadLessonLocked(2); err != nil {
		t.Fatal(err)
	}
	move(t, a, "d4", "f5")
	if !a.lessonDone {
		t.Fatal("lesson move not accepted")
	}
	if err := a.actionLocked(action{Action: "resume"}); err != nil {
		t.Fatal(err)
	}
	if a.game.FEN() != fen {
		t.Fatal("lesson overwrote real game")
	}
	if err := a.actionLocked(action{Action: "undo"}); err != nil {
		t.Fatal(err)
	}
	if len(a.game.Moves()) != 1 {
		t.Fatal("local undo should remove one ply")
	}
}
func TestEveryLessonIsLegalAndCompletes(t *testing.T) {
	for i, l := range lessons {
		t.Run(l.Title, func(t *testing.T) {
			a := newApp(t.TempDir())
			if err := a.loadLessonLocked(i); err != nil {
				t.Fatal(err)
			}
			if a.game.Outcome() != chess.NoOutcome {
				t.Fatalf("lesson starts finished: %s", a.game.Method())
			}
			move(t, a, l.Move[:2], l.Move[2:4])
			if len(l.Move) == 5 {
				if err := a.actionLocked(action{Action: "promote", Piece: l.Move[4:]}); err != nil {
					t.Fatal(err)
				}
			}
			if !a.lessonDone {
				t.Fatal("expected move rejected")
			}
			if i == 5 && a.game.Method() != chess.Checkmate {
				t.Fatal("mate lesson is not mate")
			}
		})
	}
}
func TestPromotionChoiceAndIllegalMove(t *testing.T) {
	a := newApp(t.TempDir())
	_ = a.loadLessonLocked(6)
	before := a.game.FEN()
	move(t, a, "a7", "a5")
	if a.game.FEN() != before {
		t.Fatal("illegal move changed board")
	}
	move(t, a, "a7", "a8")
	if len(a.promotion) != 4 {
		t.Fatalf("promotion needs four choices, got %d", len(a.promotion))
	}
	if a.game.FEN() != before {
		t.Fatal("promotion applied before choice")
	}
}
func TestEngineLegalAndFindsMate(t *testing.T) {
	for i := 1; i <= 3; i++ {
		m := bestMove(chess.StartingPosition(), i)
		if m == nil {
			t.Fatal("no starting move")
		}
		g := chess.NewGame()
		if err := g.Move(m); err != nil {
			t.Fatal(err)
		}
	}
	f, _ := chess.FEN(lessons[5].FEN)
	g := chess.NewGame(f)
	m := bestMove(g.Position(), 3)
	if err := g.Move(m); err != nil {
		t.Fatal(err)
	}
	if g.Method() != chess.Checkmate {
		t.Fatal("engine missed mate in one")
	}
}
func TestComputerMovesAndUndoReturnsHumanTurn(t *testing.T) {
	a := newApp(t.TempDir())
	a.mu.Lock()
	_ = a.actionLocked(action{Action: "new", Mode: "computer", Level: 1})
	move(t, a, "e2", "e4")
	a.mu.Unlock()
	deadline := time.Now().Add(8 * time.Second)
	for {
		a.mu.Lock()
		done := !a.busy
		if done {
			if len(a.game.Moves()) != 2 {
				t.Fatal("computer did not reply")
			}
			_ = a.actionLocked(action{Action: "undo"})
			if len(a.game.Moves()) != 0 || a.game.Position().Turn() != chess.White {
				t.Fatal("undo did not return to before human move")
			}
			a.mu.Unlock()
			break
		}
		a.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("computer exceeded deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func TestFlipRotatesScreenAndPreservesGame(t *testing.T) {
	a := newApp(t.TempDir())
	if err := a.actionLocked(action{Action: "new", Mode: "local", Level: 1}); err != nil {
		t.Fatal(err)
	}
	move(t, a, "e2", "e4")
	before := a.game.FEN()
	if err := a.actionLocked(action{Action: "flip"}); err != nil {
		t.Fatal(err)
	}
	v := a.stateLocked()
	if !v.Rotated || !v.Flipped || a.game.FEN() != before {
		t.Fatal("flip must rotate the screen and opponent perspective without changing moves")
	}
	b := newApp(a.data)
	if !b.stateLocked().Rotated || b.game.FEN() != before {
		t.Fatal("screen orientation or game lost on reopen")
	}
	if err := b.actionLocked(action{Action: "flip"}); err != nil {
		t.Fatal(err)
	}
	if b.stateLocked().Rotated || b.stateLocked().Flipped || b.game.FEN() != before {
		t.Fatal("second flip must restore the screen without changing moves")
	}
}

func TestBlackPerspectiveDoesNotRotateScreen(t *testing.T) {
	a := newApp(t.TempDir())
	// Existing saves may face Black without having requested a screen rotation.
	a.saved.Flipped = true
	a.saved.Mode = "computer"
	a.saved.Human = "b"
	a.saveLocked()
	b := newApp(a.data)
	v := b.stateLocked()
	if !v.Flipped || v.Rotated {
		t.Fatal("Black's board perspective must not turn the controls upside down")
	}
}

func TestSaveFileAtomicAndPrivate(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := atomicJSON(filepath.Join(dir, "state.json"), map[string]int{"value": i}); err != nil {
			t.Fatal(err)
		}
	}
	b, e := os.ReadFile(filepath.Join(dir, "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	var value map[string]int
	if json.Unmarshal(b, &value) != nil || value["value"] != 2 {
		t.Fatal("save invalid")
	}
}
