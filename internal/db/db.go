package db

import (
    "fmt"
    "os"

    _ "github.com/go-sql-driver/mysql"
    "github.com/jmoiron/sqlx"
)

type Config struct {
    Host string
    Port string
    User string
    Password string
    Name string
}

func LoadConfig() Config {
    return Config{
        Host: os.Getenv("DB_HOST"),
        Port: os.Getenv("DB_PORT"),
        User: os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASSWORD"),
        Name: os.Getenv("DB_NAME"),
    }
}

func Open(cfg Config) (*sqlx.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
    return sqlx.Open("mysql", dsn)
}
