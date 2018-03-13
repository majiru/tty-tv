package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Screen struct {
	Rows []string `json:"rows"`
}

func main() {
	// 80x20 empty screen LOL
	screen := new(Screen)
	screen.Rows = []string{
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
		"................................................................................",
	}

	http.HandleFunc("/api/screen", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := json.Marshal(screen)
		fmt.Fprintf(w, "%s", bytes)
	})

	// serve static files on `/`
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
