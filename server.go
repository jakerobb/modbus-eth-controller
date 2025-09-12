package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func StartServer() {
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		program, err := ParseProgram(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse program: %v", err), http.StatusInternalServerError)
			return
		}

		if err := program.Run(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to run program: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
