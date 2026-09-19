package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

type distributionRequest struct {
	Participants []string `json:"participants"`
	Variants     []string `json:"variants"`
}

type distributionResponse struct {
	Assignments []Assignment `json:"assignments"`
}

func main() {
	log.SetFlags(0)
	participantsPath := flag.String("participants", "", "путь к TXT-файлу с участниками")
	variantsPath := flag.String("variants", "", "путь к TXT-файлу с тем, что нужно раздать")
	outputPath := flag.String("output", "", "путь для результата CSV (по умолчанию stdout)")
	flag.Parse()

	if flag.NArg() != 0 {
		log.Fatal("неизвестные позиционные аргументы; используйте -participants и -variants")
	}
	if *participantsPath != "" || *variantsPath != "" || *outputPath != "" {
		if *participantsPath == "" || *variantsPath == "" {
			log.Fatal("для консольного режима укажите оба флага: -participants и -variants")
		}
		assignments, err := distributeTextFiles(*participantsPath, *variantsPath)
		if err != nil {
			log.Fatal(err)
		}

		output := os.Stdout
		if *outputPath != "" {
			output, err = os.Create(*outputPath)
			if err != nil {
				log.Fatalf("не удалось создать файл результата: %v", err)
			}
			defer output.Close()
		}
		if err := writeAssignmentsCSV(output, assignments); err != nil {
			log.Fatalf("не удалось записать результат: %v", err)
		}
		return
	}

	if err := serve(); err != nil {
		log.Fatal(err)
	}
}

func serve() error {
	assets, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/distribute", distributeHandler)
	mux.Handle("/", http.FileServer(http.FS(assets)))

	addr := ":6969"
	log.Printf("who-gets-what доступен на http://localhost%s", addr)
	return http.ListenAndServe(addr, securityHeaders(mux))
}

func distributeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var req distributionRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := dec.Decode(&req); err != nil {
		writeError(w, "Некорректный запрос.", http.StatusBadRequest)
		return
	}
	req.Participants = clean(req.Participants)
	req.Variants = clean(req.Variants)
	if len(req.Participants) == 0 {
		writeError(w, "Добавьте хотя бы одного участника.", http.StatusBadRequest)
		return
	}
	if len(req.Variants) == 0 {
		writeError(w, "Добавьте хотя бы один пункт во второй список.", http.StatusBadRequest)
		return
	}

	assignments, err := Distribute(req.Participants, req.Variants)
	if err != nil {
		if errors.Is(err, errEmptyInput) {
			writeError(w, "Списки не должны быть пустыми.", http.StatusBadRequest)
			return
		}
		writeError(w, "Не удалось получить безопасную случайность.", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(distributionResponse{Assignments: assignments})
}

func clean(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, message)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
