package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

type task struct {
	Id   int    `json:"id"`
	Desc string `json:"desc"`
}

var db *sql.DB

func connect() {
	connStr := "host=localhost port=5432 user=postgres password=rx dbname=todo sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	fmt.Println("Connected to the database successfully!")
}
func main() {
	connect()
	defer db.Close()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", add)
		r.Get("/", list)
		r.Put("/", update)
		r.Delete("/", delete)
	})
	fmt.Println("Server is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}
func add(w http.ResponseWriter, r *http.Request) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Desc == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	//p, err := db.Query(`SELECT COALESCE(MIN(t1.id + 1), 1) AS missing_id FROM "Tasks" t1 LEFT JOIN "Tasks" t2 ON t1.id + 1 = t2.id WHERE t2.id IS NULL`)
	p, err := db.Query(`SELECT 
CASE
WHEN (SELECT id FROM "Tasks" WHERE id=1) IS NULL THEN 1
ELSE
(select coalesce(min(t1.id +1),1) from "Tasks" t1 left join "Tasks" t2 on t1.id +1 =t2.id  where t2.id is null)
END
`)
	if err != nil {
		http.Error(w, "Error while generating id", http.StatusInternalServerError)
		//log.Fatal(err)
		return
	}
	p.Next()
	if err := p.Scan(&newTask.Id); err != nil {
		log.Fatal(err)
	}

	result, err := db.Exec(`INSERT INTO "Tasks" (id,description) VALUES ($1,$2)`, newTask.Id, newTask.Desc)
	if err != nil {
		http.Error(w, "Error while adding task", http.StatusInternalServerError)
		return
		//log.Fatal(err)
	}
	fmt.Println(result)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Task added successfully", "task": newTask})
}
func list(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, description FROM "Tasks" ORDER BY id ASC`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var tasks []task
	for rows.Next() {
		var t task
		if err := rows.Scan(&t.Id, &t.Desc); err != nil {
			log.Fatal(err)
			return
		}
		tasks = append(tasks, t)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if len(tasks) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "No tasks found", "count": 0, "tasks": []task{}})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks, "message": "success", "count": len(tasks)})
}
func update(w http.ResponseWriter, r *http.Request) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Id <= 0 || newTask.Desc == "" {
		http.Error(w, "Invalid Id or Description", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`UPDATE "Tasks" SET description=$2 WHERE id=$1`, newTask.Id, newTask.Desc)
	if err != nil {
		http.Error(w, "Error while updating task", http.StatusInternalServerError)
		return
	}
	//fmt.Println(result)
	RowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
		//http.Error(w, "Database error", http.StatusInternalServerError)
	}
	if RowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Task Updated successfully", "task": newTask})
}
func delete(w http.ResponseWriter, r *http.Request) {
	var newTask task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil || newTask.Id <= 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`DELETE FROM "Tasks" WHERE id=$1`, newTask.Id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Fatal(err)
	}
	RowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Fatal(err)
	}
	if RowsAffected == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Task not found!"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task has been deleted successfully!"})
}
