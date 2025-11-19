package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"power4/game"
)

var currentGame *game.Game

func main() {
	http.HandleFunc("/auth", handleAuth)       // Page d'authentification (login / register)
	http.HandleFunc("/", handleIndex)          // Page d'accueil
	http.HandleFunc("/start", handleStart)     // Démarrer une nouvelle partie (via bouton)
	http.HandleFunc("/choose", handleChoose)   // Choix des ballons/tokens
	http.HandleFunc("/create", handleCreate)   // Créer la partie avec les ballons
	http.HandleFunc("/game", handleGame)       // Page du jeu
	http.HandleFunc("/play", handlePlay)       // Action de jouer
	http.HandleFunc("/restart", handleRestart) // Rejouer

	// Fichiers statiques (CSS, vidéos, etc.)
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("✅ Serveur lancé sur http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Lorsqu'on visite la racine, s'assurer d'afficher la page d'accueil
	// et réinitialiser la partie en mémoire pour éviter d'afficher le plateau
	// si une partie précédente était en cours.
	currentGame = nil
	log.Printf("GET %s from %s — serving index.html", r.URL.Path, r.RemoteAddr)
	renderTemplate(w, "templates/index.html", nil)
}

// handleAuth sert la page d'authentification (login / register)
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("GET %s from %s — serving auth.html", r.URL.Path, r.RemoteAddr)
	renderTemplate(w, "templates/auth.html", nil)
}

func handleGame(w http.ResponseWriter, r *http.Request) {
	// Si aucune partie n'a été démarrée depuis la page d'accueil,
	// ne pas créer automatiquement une partie — rediriger vers l'accueil.
	if currentGame == nil {
		log.Printf("GET %s from %s — no currentGame, redirecting to /", r.URL.Path, r.RemoteAddr)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	log.Printf("GET %s from %s — serving PAGE1.html", r.URL.Path, r.RemoteAddr)
	// pass a small wrapper so templates access fields reliably
	data := struct {
		Game *game.Game
	}{Game: currentGame}
	renderTemplate(w, "templates/PAGE1.html", data)
}

// handleStart crée une nouvelle partie puis redirige vers /game.
func handleStart(w http.ResponseWriter, r *http.Request) {
	// GET -> afficher le formulaire de choix (solo / deux joueurs)
	if r.Method == http.MethodGet {
		renderTemplate(w, "templates/start.html", nil)
		return
	}

	// POST -> créer la partie selon le formulaire
	if r.Method == http.MethodPost {
		mode := r.FormValue("mode")
		name1 := r.FormValue("name1")
		name2 := r.FormValue("name2")
		if mode == "solo" {
			if name1 == "" {
				name1 = "Joueur"
			}
			name2 = "BOT"
		} else {
			if name1 == "" {
				name1 = "Joueur 1"
			}
			if name2 == "" {
				name2 = "Joueur 2"
			}
		}
		currentGame = game.NewGameWithNames(name1, name2)
		http.Redirect(w, r, "/game", http.StatusSeeOther)
		return
	}

	// autres méthodes non autorisées
	http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
}

// handleChoose affiche la page qui permet de choisir le ballon pour chaque joueur.
func handleChoose(w http.ResponseWriter, r *http.Request) {
	// accept query params name1, name2, mode
	name1 := r.URL.Query().Get("name1")
	name2 := r.URL.Query().Get("name2")
	mode := r.URL.Query().Get("mode")
	difficulty := r.URL.Query().Get("difficulty")
	data := struct {
		Name1      string
		Name2      string
		Mode       string
		Difficulty string
	}{Name1: name1, Name2: name2, Mode: mode, Difficulty: difficulty}
	renderTemplate(w, "templates/choose.html", data)
}

// handleCreate reçoit les sélections de ballons et crée la partie.
func handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	name1 := r.FormValue("name1")
	name2 := r.FormValue("name2")
	ball1 := r.FormValue("ball1")
	ball2 := r.FormValue("ball2")
	mode := r.FormValue("mode")
	difficulty := r.FormValue("difficulty")
	if mode == "solo" {
		if name1 == "" {
			name1 = "Joueur"
		}
		if name2 == "" {
			name2 = "BOT"
		}
	} else {
		if name1 == "" {
			name1 = "Joueur 1"
		}
		if name2 == "" {
			name2 = "Joueur 2"
		}
	}
	currentGame = game.NewGameWithNames(name1, name2)
	currentGame.Mode = mode
	currentGame.AIDifficulty = difficulty
	currentGame.PlayerBalls[0] = ball1
	currentGame.PlayerBalls[1] = ball2
	// redirige vers la page du jeu
	http.Redirect(w, r, "/game", http.StatusSeeOther)
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		colStr := r.FormValue("column")
		col, err := strconv.Atoi(colStr)
		if err == nil {
			currentGame.PlayMove(col)
			// Si solo mode et c'est au bot de jouer, calculer et jouer son coup
			if currentGame != nil && currentGame.Mode == "solo" && currentGame.Winner == 0 && currentGame.CurrentPlayer == 2 {
				aiCol := currentGame.AIPickMove()
				if aiCol >= 0 {
					currentGame.PlayMove(aiCol)
				}
			}
		}
	}
	http.Redirect(w, r, "/game", http.StatusSeeOther)
}

func handleRestart(w http.ResponseWriter, r *http.Request) {
	currentGame = game.NewGame()
	http.Redirect(w, r, "/game", http.StatusSeeOther)
}

func renderTemplate(w http.ResponseWriter, filepath string, data interface{}) {
	tmpl, err := template.ParseFiles(filepath)
	if err != nil {
		http.Error(w, "Erreur template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Erreur exécution: "+err.Error(), http.StatusInternalServerError)
	}
}
