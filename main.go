package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Write([]byte("Backend is running"))
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Rank     int    `json:"rank"`
}

var (
	users []User
	mu    sync.RWMutex
)

func seedUsers(n int) {
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= n; i++ {
		users = append(users, User{
			ID:       i,
			Username: "user_" + strconv.Itoa(i),
			Rating:   rand.Intn(4901) + 100,
		})
	}
}

func calculateRanks() []User {
	mu.RLock()
	defer mu.RUnlock()

	sorted := make([]User, len(users))
	copy(sorted, users)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rating > sorted[j].Rating
	})

	rank := 1
	for i := 0; i < len(sorted); i++ {
		if i > 0 && sorted[i].Rating < sorted[i-1].Rating {
			rank = i + 1
		}
		sorted[i].Rank = rank
	}
	return sorted
}

func leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	leaderboard := calculateRanks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard[:100])
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	query := strings.ToLower(r.URL.Query().Get("query"))
	if query == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}

	leaderboard := calculateRanks()
	var result []User

	for _, user := range leaderboard {
		if strings.Contains(strings.ToLower(user.Username), query) {
			result = append(result, user)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func updateRandomHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	for i := 0; i < 50; i++ {
		idx := rand.Intn(len(users))
		users[idx].Rating = rand.Intn(4901) + 100
	}
	w.Write([]byte("Ratings updated"))
}

func main() {
	seedUsers(10000)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/leaderboard", leaderboardHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/update-random", updateRandomHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
