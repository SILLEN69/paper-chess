package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/notnil/chess"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type saveGame struct {
	Initial string   `json:"initial"`
	Moves   []string `json:"moves"`
	Mode    string   `json:"mode"`
	Level   int      `json:"level"`
	Human   string   `json:"human"`
	Guide   bool     `json:"guide"`
	Flipped bool     `json:"flipped"`
}
type squareView struct {
	Square string `json:"square"`
	Piece  string `json:"piece"`
	Color  string `json:"color"`
}
type view struct {
	Board      []squareView `json:"board"`
	Turn       string       `json:"turn"`
	Status     string       `json:"status"`
	Mode       string       `json:"mode"`
	Level      int          `json:"level"`
	Human      string       `json:"human"`
	Guide      bool         `json:"guide"`
	Flipped    bool         `json:"flipped"`
	Selected   string       `json:"selected"`
	Legal      []string     `json:"legal"`
	Last       []string     `json:"last"`
	Hint       []string     `json:"hint"`
	Moves      []string     `json:"moves"`
	Message    string       `json:"message"`
	Busy       bool         `json:"busy"`
	Finished   bool         `json:"finished"`
	Check      string       `json:"check"`
	Promotion  bool         `json:"promotion"`
	Lesson     int          `json:"lesson"`
	LessonData *lesson      `json:"lessonData"`
	LessonDone bool         `json:"lessonDone"`
	Completed  int          `json:"completed"`
	Revision   int          `json:"revision"`
}
type app struct {
	mu         sync.Mutex
	game       *chess.Game
	saved      saveGame
	selected   string
	hint       []string
	message    string
	busy       bool
	revision   int
	generation int
	lesson     int
	lessonDone bool
	completed  int
	promotion  []*chess.Move
	data       string
	canQuit    bool
}

func gameFromSave(saved saveGame) (*chess.Game, error) {
	options := []func(*chess.Game){}
	if saved.Initial != "" {
		f, err := chess.FEN(saved.Initial)
		if err != nil {
			return nil, err
		}
		options = append(options, f)
	}
	game := chess.NewGame(options...)
	for _, uci := range saved.Moves {
		m, err := (chess.UCINotation{}).Decode(game.Position(), uci)
		if err != nil {
			return nil, err
		}
		if err = game.Move(m); err != nil {
			return nil, err
		}
	}
	return game, nil
}
func newApp(data string) *app {
	a := &app{data: data, lesson: -1}
	a.saved = saveGame{Mode: "computer", Level: 1, Human: "w", Guide: true}
	if content, err := os.ReadFile(filepath.Join(data, "game.json")); err == nil {
		var s saveGame
		if json.Unmarshal(content, &s) == nil {
			if g, e := gameFromSave(s); e == nil {
				a.saved = s
				a.game = g
			} else {
				a.message = "The previous save could not be read. A new board is ready."
			}
		}
	}
	if a.game == nil {
		a.game = chess.NewGame()
	}
	if a.saved.Level < 1 || a.saved.Level > 3 {
		a.saved.Level = 1
	}
	if a.saved.Mode != "local" {
		a.saved.Mode = "computer"
	}
	if a.saved.Human != "b" {
		a.saved.Human = "w"
	}
	if content, err := os.ReadFile(filepath.Join(data, "progress.json")); err == nil {
		_ = json.Unmarshal(content, &a.completed)
	}
	return a
}
func atomicJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".save-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}
func (a *app) saveLocked() {
	if a.lesson >= 0 {
		return
	}
	if err := atomicJSON(filepath.Join(a.data, "game.json"), a.saved); err != nil {
		a.message = "Could not save the game: " + err.Error()
	}
}
func (a *app) changed() { a.revision++ }
func (a *app) stateLocked() view {
	p := a.game.Position()
	if a.hint == nil {
		a.hint = []string{}
	}
	v := view{Mode: a.saved.Mode, Level: a.saved.Level, Human: a.saved.Human, Guide: a.saved.Guide, Flipped: a.saved.Flipped, Turn: p.Turn().Name(), Selected: a.selected, Hint: a.hint, Message: a.message, Busy: a.busy, Lesson: a.lesson, LessonDone: a.lessonDone, Completed: a.completed, Revision: a.revision, Promotion: len(a.promotion) > 0, Board: []squareView{}, Legal: []string{}, Last: []string{}, Moves: []string{}}
	for rank := 7; rank >= 0; rank-- {
		for file := 0; file < 8; file++ {
			sq := chess.NewSquare(chess.File(file), chess.Rank(rank))
			piece := p.Board().Piece(sq)
			color := ""
			if piece.Color() != chess.NoColor {
				color = piece.Color().String()
			}
			v.Board = append(v.Board, squareView{sq.String(), piece.Type().String(), color})
		}
	}
	for _, m := range a.game.ValidMoves() {
		if m.S1().String() == a.selected {
			v.Legal = append(v.Legal, m.S2().String())
		}
	}
	positions := a.game.Positions()
	for i, m := range a.game.Moves() {
		v.Moves = append(v.Moves, (chess.AlgebraicNotation{}).Encode(positions[i], m))
	}
	ms := a.game.Moves()
	if len(ms) > 0 {
		m := ms[len(ms)-1]
		v.Last = []string{m.S1().String(), m.S2().String()}
	}
	v.Status = p.Turn().Name() + " to move"
	if a.busy {
		v.Status = "Thinking…"
	}
	if a.game.Outcome() != chess.NoOutcome {
		v.Finished = true
		switch a.game.Outcome() {
		case chess.WhiteWon:
			v.Status = "White wins"
		case chess.BlackWon:
			v.Status = "Black wins"
		default:
			v.Status = "Draw"
		}
		v.Status += " · " + a.game.Method().String()
	}
	if p.Status() == chess.Checkmate || lastCheck(a.game) {
		for sq, piece := range p.Board().SquareMap() {
			if piece.Type() == chess.King && piece.Color() == p.Turn() {
				v.Check = sq.String()
			}
		}
		if !v.Finished && !a.busy {
			v.Status += " · check"
		}
	}
	if a.lesson >= 0 {
		l := lessons[a.lesson]
		v.LessonData = &l
		v.Mode = "learn"
		v.Flipped = false
		v.Guide = true
		if a.lessonDone {
			v.Status = "Lesson complete"
		} else {
			v.Status = "Lesson " + fmt.Sprint(a.lesson+1) + " of " + fmt.Sprint(len(lessons))
		}
	}
	return v
}
func lastCheck(game *chess.Game) bool {
	m := game.Moves()
	if len(m) > 0 {
		return m[len(m)-1].HasTag(chess.Check)
	}
	return false
}
func squareOf(text string) (chess.Square, bool) {
	if len(text) != 2 || text[0] < 'a' || text[0] > 'h' || text[1] < '1' || text[1] > '8' {
		return chess.NoSquare, false
	}
	return chess.NewSquare(chess.File(text[0]-'a'), chess.Rank(text[1]-'1')), true
}
func (a *app) applyLocked(m *chess.Move) error {
	if a.lesson >= 0 && m.String() != lessons[a.lesson].Move {
		a.message = "Try the move described above. Tap Hint if you need a hand."
		a.changed()
		return nil
	}
	before := a.game.Position()
	if err := a.game.Move(m); err != nil {
		return err
	}
	a.saved.Moves = append(a.saved.Moves, m.String())
	a.selected = ""
	a.promotion = nil
	a.hint = nil
	a.message = explainMove(before, m)
	if a.lesson >= 0 {
		a.lessonDone = true
		a.message = lessons[a.lesson].Why
		if a.completed < a.lesson+1 {
			a.completed = a.lesson + 1
			_ = atomicJSON(filepath.Join(a.data, "progress.json"), a.completed)
		}
	} else {
		a.saveLocked()
	}
	a.generation++
	a.changed()
	a.startComputerLocked()
	return nil
}
func explainMove(pos *chess.Position, m *chess.Move) string {
	piece := pos.Board().Piece(m.S1())
	name := map[chess.PieceType]string{chess.Pawn: "pawn", chess.Knight: "knight", chess.Bishop: "bishop", chess.Rook: "rook", chess.Queen: "queen", chess.King: "king"}[piece.Type()]
	if m.HasTag(chess.KingSideCastle) || m.HasTag(chess.QueenSideCastle) {
		return "Castling brings the king toward safety and connects your rooks."
	}
	if m.Promo() != chess.NoPieceType {
		return "The pawn promotes. Choose the piece that best fits the position."
	}
	if m.HasTag(chess.EnPassant) {
		return "En passant: the pawn captures diagonally behind the pawn that just advanced two squares."
	}
	if m.HasTag(chess.Capture) {
		return "The " + name + " captures on " + m.S2().String() + ". Before capturing, check whether it can be taken back."
	}
	if m.HasTag(chess.Check) {
		return "Check. The other player must get the king out of attack."
	}
	if (piece.Type() == chess.Knight || piece.Type() == chess.Bishop) && (m.S1().Rank() == chess.Rank1 || m.S1().Rank() == chess.Rank8) {
		return "Developing a minor piece helps control the centre and prepares castling."
	}
	if piece.Type() == chess.Pawn && (m.S2().File() == chess.FileD || m.S2().File() == chess.FileE) {
		return "A central pawn claims space and opens lines for your pieces."
	}
	return "Before your next move, look for checks, captures, and threats on both sides."
}
func (a *app) startComputerLocked() {
	if a.lesson >= 0 || a.busy || a.saved.Mode != "computer" || a.game.Outcome() != chess.NoOutcome || a.game.Position().Turn().String() == a.saved.Human {
		return
	}
	a.busy = true
	generation := a.generation
	fen := a.game.FEN()
	level := a.saved.Level
	a.changed()
	go func() {
		f, _ := chess.FEN(fen)
		move := bestMove(chess.NewGame(f).Position(), level)
		a.mu.Lock()
		defer a.mu.Unlock()
		if generation != a.generation {
			return
		}
		a.busy = false
		if move != nil {
			_ = a.applyLocked(move)
		} else {
			a.changed()
		}
	}()
}

type action struct {
	Action string `json:"action"`
	Square string `json:"square"`
	Mode   string `json:"mode"`
	Level  int    `json:"level"`
	Human  string `json:"human"`
	Index  int    `json:"index"`
	Piece  string `json:"piece"`
}

func (a *app) actionLocked(cmd action) error {
	switch cmd.Action {
	case "tap":
		if a.busy || a.game.Outcome() != chess.NoOutcome || (a.lesson >= 0 && a.lessonDone) {
			return nil
		}
		sq, ok := squareOf(cmd.Square)
		if !ok {
			return errors.New("invalid square")
		}
		if a.lesson < 0 && a.saved.Mode == "computer" && a.game.Position().Turn().String() != a.saved.Human {
			return nil
		}
		if a.selected != "" {
			choices := []*chess.Move{}
			for _, m := range a.game.ValidMoves() {
				if m.S1().String() == a.selected && m.S2() == sq {
					choices = append(choices, m)
				}
			}
			if len(choices) > 0 {
				if choices[0].Promo() != chess.NoPieceType {
					a.promotion = choices
					a.changed()
					return nil
				}
				return a.applyLocked(choices[0])
			}
		}
		p := a.game.Position().Board().Piece(sq)
		a.hint = nil
		a.promotion = nil
		if p.Color() == a.game.Position().Turn() {
			if a.selected == cmd.Square {
				a.selected = ""
			} else {
				a.selected = cmd.Square
				a.message = pieceHelp(p.Type())
			}
		} else {
			a.selected = ""
			a.message = "Tap one of the pieces whose turn it is, then a marked destination."
		}
		a.changed()
	case "promote":
		for _, m := range a.promotion {
			if m.Promo().String() == cmd.Piece {
				return a.applyLocked(m)
			}
		}
		return errors.New("choose a promotion piece")
	case "new":
		if cmd.Mode != "local" && cmd.Mode != "computer" {
			return errors.New("invalid game mode")
		}
		if cmd.Level < 1 || cmd.Level > 3 {
			cmd.Level = 1
		}
		if cmd.Human != "b" {
			cmd.Human = "w"
		}
		a.generation++
		a.busy = false
		a.lesson = -1
		a.lessonDone = false
		a.game = chess.NewGame()
		a.saved = saveGame{Mode: cmd.Mode, Level: cmd.Level, Human: cmd.Human, Guide: true, Flipped: cmd.Human == "b"}
		a.selected = ""
		a.hint = nil
		a.promotion = nil
		a.message = "Tap a piece to see its legal moves."
		a.saveLocked()
		a.changed()
		a.startComputerLocked()
	case "undo":
		if a.lesson >= 0 {
			return a.loadLessonLocked(a.lesson)
		}
		a.generation++
		a.busy = false
		n := len(a.saved.Moves)
		if n > 0 {
			n--
			if a.saved.Mode == "computer" {
				for n > 0 {
					copy := a.saved
					copy.Moves = copy.Moves[:n]
					game, e := gameFromSave(copy)
					if e != nil {
						break
					}
					if game.Position().Turn().String() == a.saved.Human {
						break
					}
					n--
				}
			}
			a.saved.Moves = a.saved.Moves[:n]
			g, e := gameFromSave(a.saved)
			if e != nil {
				return e
			}
			a.game = g
		}
		a.selected = ""
		a.promotion = nil
		a.hint = nil
		a.message = "Move taken back. Try another idea."
		a.saveLocked()
		a.changed()
		a.startComputerLocked()
	case "hint":
		if a.busy {
			return nil
		}
		if a.lesson >= 0 {
			m := lessons[a.lesson].Move
			a.hint = []string{m[:2], m[2:4]}
			a.selected = m[:2]
			a.message = "Try " + m[:2] + " to " + m[2:4] + "."
			a.changed()
			return nil
		}
		if a.game.Outcome() != chess.NoOutcome {
			return nil
		}
		a.busy = true
		generation := a.generation
		fen := a.game.FEN()
		a.changed()
		go func() {
			f, _ := chess.FEN(fen)
			p := chess.NewGame(f).Position()
			m := bestMove(p, 3)
			a.mu.Lock()
			defer a.mu.Unlock()
			if generation != a.generation {
				return
			}
			a.busy = false
			if m != nil {
				a.hint = []string{m.S1().String(), m.S2().String()}
				a.selected = m.S1().String()
				a.message = "Try " + (chess.AlgebraicNotation{}).Encode(p, m) + ". " + explainMove(p, m)
			}
			a.changed()
		}()
	case "flip":
		a.saved.Flipped = !a.saved.Flipped
		a.saveLocked()
		a.changed()
	case "guide":
		a.saved.Guide = !a.saved.Guide
		a.saveLocked()
		a.changed()
	case "lesson":
		return a.loadLessonLocked(cmd.Index)
	case "resume":
		a.generation++
		a.busy = false
		a.lesson = -1
		a.lessonDone = false
		a.selected = ""
		a.hint = nil
		a.promotion = nil
		data, err := os.ReadFile(filepath.Join(a.data, "game.json"))
		if err == nil {
			var saved saveGame
			if json.Unmarshal(data, &saved) == nil {
				if g, e := gameFromSave(saved); e == nil {
					a.saved = saved
					a.game = g
				}
			}
		} else {
			a.saved = saveGame{Mode: "computer", Level: 1, Human: "w", Guide: true}
			a.game = chess.NewGame()
		}
		a.message = "Your game is ready."
		a.changed()
		a.startComputerLocked()
	default:
		return errors.New("unknown action")
	}
	return nil
}
func (a *app) loadLessonLocked(index int) error {
	if index < 0 || index >= len(lessons) {
		return errors.New("unknown lesson")
	}
	a.generation++
	a.busy = false
	f, e := chess.FEN(lessons[index].FEN)
	if e != nil {
		return e
	}
	a.game = chess.NewGame(f)
	a.lesson = index
	a.lessonDone = false
	a.selected = ""
	a.hint = nil
	a.promotion = nil
	a.message = "Tap a piece, then its destination."
	a.changed()
	return nil
}
func pieceHelp(t chess.PieceType) string {
	return map[chess.PieceType]string{chess.Pawn: "Pawns move forward and capture diagonally. The dots show legal destinations.", chess.Knight: "Knights jump in an L: two squares one way, then one sideways.", chess.Bishop: "Bishops move along clear diagonals, staying on one square colour.", chess.Rook: "Rooks move along clear ranks and files, any number of squares.", chess.Queen: "The queen moves along clear ranks, files, and diagonals.", chess.King: "The king moves one square without entering attack. Castling may also be available."}[t]
}
func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(a.stateLocked())
	})
	mux.HandleFunc("/api/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON POST required", 405)
			return
		}
		var cmd action
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&cmd); err != nil {
			http.Error(w, "invalid request", 400)
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if err := a.actionLocked(cmd); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(a.stateLocked())
	})
	mux.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		w.WriteHeader(202)
		if a.canQuit {
			go func() {
				time.Sleep(150 * time.Millisecond)
				if err := exec.Command("systemd-run", "--quiet", "--collect", "--unit=paper-chess-close", "/home/root/paper-chess/session", "stop").Run(); err != nil {
					log.Print(err)
				}
			}()
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "null" && origin != "http://127.0.0.1:9148" {
			http.Error(w, "origin not allowed", 403)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func main() {
	clock := flag.Bool("monotonic-micros", false, "read the tablet monotonic clock for safe interface switching")
	watch := flag.Bool("watch-buttons", false, "watch the tablet power shortcut without changing the stock interface")
	tap := flag.String("diagnostic-tap", "", "explicit native touchscreen diagnostic x,y")
	data := flag.String("data", "/home/root/.local/share/paper-chess", "save directory")
	canQuit := flag.Bool("tablet", false, "enable return-to-notes action")
	flag.Parse()
	if *clock {
		value, err := monotonicMicros()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value)
		return
	}
	if *watch {
		log.Fatal(watchButtons())
		return
	}
	if *tap != "" {
		var x, y int
		if _, e := fmt.Sscanf(*tap, "%d,%d", &x, &y); e != nil {
			log.Fatal(e)
		}
		if e := diagnosticTap(x, y); e != nil {
			log.Fatal(e)
		}
		return
	}
	a := newApp(*data)
	a.canQuit = *canQuit
	a.mu.Lock()
	a.startComputerLocked()
	a.mu.Unlock()
	server := http.Server{Addr: "127.0.0.1:9148", Handler: a.handler(), ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 30 * time.Second}
	log.Fatal(server.ListenAndServe())
}
