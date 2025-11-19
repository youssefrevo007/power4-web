package game

import (
	"math"
	"math/rand"
	"time"
)

type Game struct {
	Board         [][]int   // 6x7 : 0 = vide, 1 = joueur1, 2 = joueur2
	CurrentPlayer int       // joueur actif : 1 ou 2
	Winner        int       // 0 = pas de gagnant, 1 ou 2 si quelqu’un gagne
	PlayerNames   [2]string // noms des joueurs (index 0 -> joueur1, index1 -> joueur2)
	PlayerBalls   [2]string // choix de balle/token pour chaque joueur
	Mode          string    // 'two' or 'solo' (bot)
	AIDifficulty  string    // difficulty when playing vs bot: 'debutant','amateur','expert'
}

// Crée une nouvelle partie
func NewGame() *Game {
	return NewGameWithNames("Joueur 1", "Joueur 2")
}

// NewGameWithNames crée une nouvelle partie en initialisant les noms des joueurs.
func NewGameWithNames(name1, name2 string) *Game {
	board := make([][]int, 6)
	for i := range board {
		board[i] = make([]int, 7)
	}
	g := &Game{
		Board:         board,
		CurrentPlayer: 1,
	}
	if name1 == "" {
		name1 = "Joueur 1"
	}
	if name2 == "" {
		name2 = "Joueur 2"
	}
	g.PlayerNames[0] = name1
	g.PlayerNames[1] = name2
	return g
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// --- Helpers for AI ---

// ValidMoves retourne les colonnes valides (non pleines)
func (g *Game) ValidMoves() []int {
	moves := []int{}
	cols := len(g.Board[0])
	for c := 0; c < cols; c++ {
		if g.Board[0][c] == 0 {
			moves = append(moves, c)
		}
	}
	return moves
}

// deep copy du plateau
func (g *Game) cloneBoard() [][]int {
	h := len(g.Board)
	w := len(g.Board[0])
	nb := make([][]int, h)
	for i := 0; i < h; i++ {
		nb[i] = make([]int, w)
		copy(nb[i], g.Board[i])
	}
	return nb
}

// dropPiece simule un drop sur une copie de board
func dropPiece(board [][]int, col int, player int) (int, bool) {
	for r := len(board) - 1; r >= 0; r-- {
		if board[r][col] == 0 {
			board[r][col] = player
			return r, true
		}
	}
	return -1, false
}

// checkWinBoard vérifie une victoire sur un plateau donné
func checkWinBoard(board [][]int, r, c, player int) bool {
	if r < 0 || c < 0 {
		return false
	}
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range directions {
		count := 1
		count += countDirectionBoard(board, r, c, d[0], d[1], player)
		count += countDirectionBoard(board, r, c, -d[0], -d[1], player)
		if count >= 4 {
			return true
		}
	}
	return false
}

func countDirectionBoard(board [][]int, r, c, dr, dc, player int) int {
	count := 0
	for {
		r += dr
		c += dc
		if r < 0 || r >= len(board) || c < 0 || c >= len(board[0]) {
			break
		}
		if board[r][c] != player {
			break
		}
		count++
	}
	return count
}

// evaluateBoard fournit un score heuristique: positif = avantage BOT (player 2)
func evaluateBoard(board [][]int) int {
	score := 0
	h := len(board)
	w := len(board[0])
	// simple window scan: for chaque fenêtre de 4 cases horizontale/vert/diag
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			for _, d := range directions {
				cnt1 := 0
				cnt2 := 0
				empty := 0
				rr, cc := r, c
				for k := 0; k < 4; k++ {
					if rr < 0 || rr >= h || cc < 0 || cc >= w {
						cnt1 = -1
						break
					}
					if board[rr][cc] == 1 {
						cnt1++
					} else if board[rr][cc] == 2 {
						cnt2++
					} else {
						empty++
					}
					rr += d[0]
					cc += d[1]
				}
				if cnt1 == -1 {
					continue
				}
				// scoring: 4 in a row -> big; 3 -> medium, 2 -> small
				if cnt2 == 4 {
					score += 100000
				} else if cnt2 == 3 && empty == 1 {
					score += 1000
				} else if cnt2 == 2 && empty == 2 {
					score += 50
				}
				if cnt1 == 4 {
					score -= 100000
				} else if cnt1 == 3 && empty == 1 {
					score -= 1000
				} else if cnt1 == 2 && empty == 2 {
					score -= 50
				}
			}
		}
	}
	return score
}

// ordre de colonnes préféré (centre first)
func preferredOrder(w int) []int {
	order := make([]int, 0, w)
	center := w / 2
	order = append(order, center)
	for i := 1; i <= center; i++ {
		if center-i >= 0 {
			order = append(order, center-i)
		}
		if center+i < w {
			order = append(order, center+i)
		}
	}
	return order
}

// minimax with alpha-beta
func minimax(board [][]int, depth int, alpha, beta int, maximizing bool) (int, int) {
	// returns score and chosen column
	// check terminal or depth
	moves := []int{}
	w := len(board[0])
	for c := 0; c < w; c++ {
		if board[0][c] == 0 {
			moves = append(moves, c)
		}
	}
	// terminal checks: win for any side
	// scan for immediate win
	for _, c := range moves {
		b := make([][]int, len(board))
		for i := range board {
			b[i] = make([]int, len(board[0]))
			copy(b[i], board[i])
		}
		r, ok := dropPiece(b, c, 2)
		if ok && checkWinBoard(b, r, c, 2) {
			return 1000000, c
		}
		// check opponent
		b2 := make([][]int, len(board))
		for i := range board {
			b2[i] = make([]int, len(board[0]))
			copy(b2[i], board[i])
		}
		r2, ok2 := dropPiece(b2, c, 1)
		if ok2 && checkWinBoard(b2, r2, c, 1) {
			return -1000000, c
		}
	}

	if depth == 0 || len(moves) == 0 {
		return evaluateBoard(board), -1
	}

	bestCol := moves[0]
	if maximizing {
		value := math.MinInt32
		order := preferredOrder(len(board[0]))
		for _, c := range order {
			if board[0][c] != 0 {
				continue
			}
			b := make([][]int, len(board))
			for i := range board {
				b[i] = make([]int, len(board[0]))
				copy(b[i], board[i])
			}
			dropPiece(b, c, 2)
			sc, _ := minimax(b, depth-1, alpha, beta, false)
			if sc > value {
				value = sc
				bestCol = c
			}
			if value > alpha {
				alpha = value
			}
			if alpha >= beta {
				break
			}
		}
		return value, bestCol
	} else {
		value := math.MaxInt32
		order := preferredOrder(len(board[0]))
		for _, c := range order {
			if board[0][c] != 0 {
				continue
			}
			b := make([][]int, len(board))
			for i := range board {
				b[i] = make([]int, len(board[0]))
				copy(b[i], board[i])
			}
			dropPiece(b, c, 1)
			sc, _ := minimax(b, depth-1, alpha, beta, true)
			if sc < value {
				value = sc
				bestCol = c
			}
			if value < beta {
				beta = value
			}
			if alpha >= beta {
				break
			}
		}
		return value, bestCol
	}
}

// AIPickMove choisit une colonne selon la difficulté
func (g *Game) AIPickMove() int {
	moves := g.ValidMoves()
	if len(moves) == 0 {
		return -1
	}
	// Débutant: random
	if g.AIDifficulty == "debutant" {
		return moves[rand.Intn(len(moves))]
	}
	// Amateur: check win, block, center, random
	if g.AIDifficulty == "amateur" {
		// try to win
		for _, c := range moves {
			b := g.cloneBoard()
			r, _ := dropPiece(b, c, 2)
			if checkWinBoard(b, r, c, 2) {
				return c
			}
		}
		// try to block opponent
		for _, c := range moves {
			b := g.cloneBoard()
			r, _ := dropPiece(b, c, 1)
			if checkWinBoard(b, r, c, 1) {
				return c
			}
		}
		// center preference
		center := len(g.Board[0]) / 2
		for _, c := range moves {
			if c == center {
				return c
			}
		}
		// else random
		return moves[rand.Intn(len(moves))]
	}
	// Expert: minimax with depth 4
	depth := 4
	_, col := minimax(g.cloneBoard(), depth, math.MinInt32, math.MaxInt32, true)
	if col < 0 {
		// fallback
		return moves[rand.Intn(len(moves))]
	}
	return col
}

// Joue un coup dans une colonne
func (g *Game) PlayMove(col int) bool {
	if col < 0 || col >= len(g.Board[0]) || g.Winner != 0 {
		return false
	}
	for row := len(g.Board) - 1; row >= 0; row-- {
		if g.Board[row][col] == 0 {
			g.Board[row][col] = g.CurrentPlayer
			if g.checkWin(row, col) {
				g.Winner = g.CurrentPlayer
			}
			g.switchPlayer()
			return true
		}
	}
	return false
}

// Change de joueur
func (g *Game) switchPlayer() {
	if g.CurrentPlayer == 1 {
		g.CurrentPlayer = 2
	} else {
		g.CurrentPlayer = 1
	}
}

// Vérifie si le dernier coup a gagné
func (g *Game) checkWin(r, c int) bool {
	player := g.Board[r][c]
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range directions {
		count := 1
		count += g.countDirection(r, c, d[0], d[1], player)
		count += g.countDirection(r, c, -d[0], -d[1], player)
		if count >= 4 {
			return true
		}
	}
	return false
}

// Compte les pions alignés dans une direction
func (g *Game) countDirection(r, c, dr, dc, player int) int {
	count := 0
	for {
		r += dr
		c += dc
		if r < 0 || r >= len(g.Board) || c < 0 || c >= len(g.Board[0]) {
			break
		}
		if g.Board[r][c] != player {
			break
		}
		count++
	}
	return count
}
