package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/joho/godotenv"

    "mobilesyncserverrk/internal/db"
    "mobilesyncserverrk/internal/handlers"
    syncsvc "mobilesyncserverrk/internal/sync"
)

func main() {
    migrate := flag.Bool("migrate", false, "run DB migrations and exit")
    flag.Parse()

    _ = godotenv.Load()

    cfg := db.LoadConfig()
    if cfg.Host == "" || cfg.User == "" || cfg.Name == "" {
        log.Fatal("missing DB config; copy .env.example to .env")
    }

    database, err := db.Open(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()

    if err := database.Ping(); err != nil {
        log.Fatal(err)
    }

    if *migrate {
        if err := db.RunMigrations(database, "migrations"); err != nil {
            log.Fatal(err)
        }
        fmt.Println("migrations applied")
        return
    }

    svc := &syncsvc.Service{DB: database}
    handler := &handlers.SyncHandler{Service: svc}

    mux := http.NewServeMux()
    mux.HandleFunc("/sync/push", handler.Push)
    mux.HandleFunc("/sync/pull", handler.Pull)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
