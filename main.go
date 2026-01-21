package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

/* -------------------- CORS -------------------- */

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

/* -------------------- MODELS -------------------- */

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Rank     int    `json:"rank"`
}

/* -------------------- GLOBAL STATE -------------------- */

var (
	users       []User
	rankedUsers []User
	mu          sync.RWMutex
)

/* -------------------- SEED USERS (10,000+) -------------------- */

func seedUsers() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
		users = append(users, User{
			ID:       i + 1,
			Username: fmt.Sprintf("user_%d", i),
			Rating:   rand.Intn(4901-100) + 100,
		})
	}
}

/* -------------------- TIE-AWARE RANK CALCULATION -------------------- */

func calculateRanks() {
	mu.Lock()
	defer mu.Unlock()

	rankedUsers = make([]User, len(users))
	copy(rankedUsers, users)

	sort.Slice(rankedUsers, func(i, j int) bool {
		return rankedUsers[i].Rating > rankedUsers[j].Rating
	})

	rank := 1
	prevRating := -1
	sameCount := 0

	for i := 0; i < len(rankedUsers); i++ {
		if rankedUsers[i].Rating == prevRating {
			rankedUsers[i].Rank = rank
			sameCount++
		} else {
			rank = rank + sameCount
			rankedUsers[i].Rank = rank
			sameCount = 1
			prevRating = rankedUsers[i].Rating
		}
	}
}

/* -------------------- BACKGROUND SCORE UPDATE -------------------- */

func startScoreSimulation() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			mu.Lock()
			for i := 0; i < 50; i++ {
				idx := rand.Intn(len(users))
				users[idx].Rating = rand.Intn(4901-100) + 100
			}
			mu.Unlock()
			calculateRanks()
		}
	}()
}

/* -------------------- HANDLERS -------------------- */

func rootHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Write([]byte("Leaderboard Backend is running"))
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rankedUsers[:100])
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	query := strings.ToLower(r.URL.Query().Get("query"))
	if query == "" {
		http.Error(w, "query parameter required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	var result []User
	for _, user := range rankedUsers {
		if strings.Contains(strings.ToLower(user.Username), query) {
			result = append(result, user)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

/* -------------------- MAIN -------------------- */

func main() {
	seedUsers()
	calculateRanks()
	startScoreSimulation()

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/leaderboard", leaderboardHandler)
	http.HandleFunc("/search", searchHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
