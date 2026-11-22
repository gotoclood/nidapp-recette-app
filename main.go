// Minimal HTTP application used by the platform acceptance runs.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	_ "github.com/lib/pq"
)

// version is what "/" and "/version" answer. Change it to prove a new build.
const version = "1.0.0"

var tableName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"app": "nidapp-recette-app", "version": version})
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": version})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	// Always fails: used to raise the error rate of the service on purpose.
	mux.HandleFunc("GET /fail", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failure on request"})
	})
	mux.HandleFunc("GET /db", handleDB)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("nidapp-recette-app %s listening on :%s", version, port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

// handleDB reads the witness table through DATABASE_URL and returns its rows.
// The table name comes from WITNESS_TABLE (default "temoin").
func handleDB(w http.ResponseWriter, r *http.Request) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "DATABASE_URL is not set"})
		return
	}
	table := os.Getenv("WITNESS_TABLE")
	if table == "" {
		table = "temoin"
	}
	if !tableName.MatchString(table) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "WITNESS_TABLE is not a valid table name"})
		return
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cannot open the database"})
		return
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("close database: %v", cerr)
		}
	}()

	rows, err := db.QueryContext(r.Context(), fmt.Sprintf(`SELECT * FROM %q ORDER BY 1 LIMIT 100`, table))
	if err != nil {
		// The driver error can quote the connection settings: log it, do not return it.
		log.Printf("query %s: %v", table, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "query failed, see the logs"})
		return
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("close rows: %v", cerr)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "cannot read the columns"})
		return
	}
	result := make([]map[string]string, 0)
	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "cannot read a row"})
			return
		}
		row := make(map[string]string, len(columns))
		for i, name := range columns {
			row[name] = values[i].String
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "cannot read the rows"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"table": table, "count": len(result), "rows": result})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}
