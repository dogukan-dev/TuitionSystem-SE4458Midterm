package main

import (
	"context"
	"dogukan-dev/tuition/db"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type App struct {
	Queries *db.Queries
	Context context.Context
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_CONNECTION"))
	if err != nil {
		fmt.Errorf("Error on pgx connection")
	}
	defer conn.Close(ctx)

	app := &App{
		Queries: db.New(conn),
		Context: ctx,
	}

	// Read schema.sql
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("cannot read schema.sql: %v", err)
	}

	// Execute schema
	_, err = conn.Exec(ctx, string(schema))
	if err != nil {
		log.Fatalf("failed to apply schema: %v", err)
	}
	log.Println("\n\nSchema applied successfully!")

	initLogger()
	defer logFile.Close()

	mux := http.NewServeMux()

	// v1 API
	v1Mux := http.NewServeMux()
	v1Mux.HandleFunc("/health", healthHandler)
	v1Mux.HandleFunc("/register", loggingMiddleware(app.registerHandler))
	v1Mux.HandleFunc("/login", loggingMiddleware(app.loginHandler))
	v1Mux.HandleFunc("/mobile/tuition", loggingMiddleware(app.QueryTuitionHandler))
	v1Mux.HandleFunc("/banking/tuition", loggingMiddleware(app.QueryTuitionHandler))
	v1Mux.HandleFunc("/admin/add-tuition", loggingMiddleware(app.addTuitionHandler))
	v1Mux.HandleFunc("/admin/add-student", loggingMiddleware(app.addStudentHandler))

	// v2 API
	v2Mux := http.NewServeMux()
	v2Mux.HandleFunc("/health", healthHandler)
	v2Mux.HandleFunc("/mobile/tuition", loggingMiddleware(app.routingMiddleware(authMiddleware(app.rateLimitMiddleware(app.QueryTuitionHandler)))))
	v2Mux.HandleFunc("/banking/tuition", loggingMiddleware(authMiddleware(app.QueryTuitionHandler)))
	v2Mux.HandleFunc("/banking/pay", loggingMiddleware(app.PayTuitionHandler))
	v2Mux.HandleFunc("/admin/add-tuition", loggingMiddleware(authMiddleware(app.addTuitionHandler)))
	v2Mux.HandleFunc("/admin/add-tuition-batch", loggingMiddleware(authMiddleware(app.addTuitionBatchHandler)))
	v2Mux.HandleFunc("/admin/unpaid-status", loggingMiddleware(authMiddleware(app.unpaidTuitionStatusHandler)))
	v2Mux.HandleFunc("/admin/add-student", loggingMiddleware(authMiddleware(app.addStudentHandler)))
	v2Mux.HandleFunc("/register", loggingMiddleware(app.registerHandler))
	v2Mux.HandleFunc("/login", loggingMiddleware(app.loginHandler))

	mux.HandleFunc("/swagger-ui", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./swagger-ui.html")
	})
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./swagger.json")
	})
	// Mount versions
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Mux))
	mux.Handle("/api/v2/", http.StripPrefix("/api/v2", v2Mux))

	port := ":" + os.Getenv("PORT")
	log.Printf("Server starting on port %s", port)
	log.Printf("Swagger documentation available at http://localhost%s/swagger.json", port)
	log.Printf("Swagger ui available at http://localhost%s/swagger-ui", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
