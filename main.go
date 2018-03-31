package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
)

const ConfigName = "config.ini"

var db *sql.DB

func main() {
	dbc := getDatabaseConfig()
	if dbc.load(ConfigName) != nil {
		dbc.save(ConfigName)
		fmt.Println("Generated default config at ", ConfigName, ". Please fill correct values.")
		os.Exit(1)
	}

	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@%s(%s:%d)/%s", dbc.Username, dbc.Password, dbc.Protocol, dbc.Hostname, dbc.Port, dbc.Database))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Could not connect to the database. Please confirm ", ConfigName, " is using the correct values.")
	}

	router := mux.NewRouter()
	router.HandleFunc("/event", GetEvents).Methods("GET")
	router.HandleFunc("/event/{id}", GetEvent).Methods("GET")

	fmt.Println("Starting web server on port 8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}

type DatabaseConfig struct {
	Hostname string `ini:"hostname"`
	Port     int    `ini:"port"`
	Protocol string `ini:"protocol"`
	Database string `ini:"database"`
	Username string `ini:"username"`
	Password string `ini:"password"`
}

func getDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Hostname: "127.0.0.1",
		Port:     3306,
		Protocol: "tcp",
		Database: "REPLACE_ME",
		Username: "REPLACE_ME",
		Password: "REPLACE_ME",
	}
}

func (dbc *DatabaseConfig) load(filename string) error {
	f, err := ini.Load(filename)
	if err != nil {
		return err
	}

	err = f.MapTo(dbc)
	if err != nil {
		return err
	}

	return nil
}

func (dbc *DatabaseConfig) save(filename string) {
	tmp := ini.Empty()
	err := ini.ReflectFrom(tmp, dbc)
	if err != nil {
		log.Fatal(err)
	}
	tmp.SaveTo(filename)
}

type Event struct {
	Id           int            `json:"id"`
	Location     string         `json:"location"`
	Department   string         `json:"department"`
	Category     string         `json:"category"`
	Priority     string         `json:"priority"`
	Description  sql.NullString `json:"description"`
	Remarks      sql.NullString `json:"remarks"`
	ReportedById int            `json:"reportedby"`
	OperativeId  sql.NullInt64  `json:"operativeid"`
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM Event")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.Id, &e.Location, &e.Department, &e.Category, &e.Priority, &e.Description, &e.Remarks, &e.ReportedById, &e.OperativeId)
		if err != nil {
			log.Fatal(err)
		}
		events = append(events, e)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(w).Encode(events)
}

func GetEvent(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Make sure ID is an Integer
	if _, err := strconv.Atoi(params["id"]); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 1, "message": "ID was not an Integer"})
		return
	}

	var e Event
	err := db.QueryRow("SELECT * FROM Event WHERE id = ?", params["id"]).Scan(&e.Id, &e.Location, &e.Department, &e.Category, &e.Priority, &e.Description, &e.Remarks, &e.ReportedById, &e.OperativeId)
	if err != nil {
		if err == sql.ErrNoRows {
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 2, "message": "No results"})
			return
		} else {
			log.Fatal(err)
		}
	}

	json.NewEncoder(w).Encode(e)
}
