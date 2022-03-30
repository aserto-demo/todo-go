package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"go-server/store"
	"go-server/structs"
)

type Todo = structs.Todo

func GetTodos(w http.ResponseWriter, r *http.Request) {
	var todos []Todo

	todos, err := store.GetTodos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todos)
	}
}

func InsertTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	jsonErr := json.NewDecoder(r.Body).Decode(&todo)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusBadRequest)
		return
	}

	err := store.InsertTodo(todo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		json.NewEncoder(w).Encode(todo)
	}
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	jsonErr := json.NewDecoder(r.Body).Decode(&todo)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusBadRequest)
		return
	}

	err := store.UpdateTodo(todo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		json.NewEncoder(w).Encode(todo)
	}
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	jsonErr := json.NewDecoder(r.Body).Decode(&todo)
	if jsonErr != nil {
		http.Error(w, jsonErr.Error(), http.StatusBadRequest)
		return
	}

	err := store.DeleteTodo(todo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(200)
	}
}

func CORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Authorization")
			return
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func Start(handler http.Handler) {
	fmt.Println("Staring server on 0.0.0.0:8080")

	srv := http.Server{
		Handler: CORS(handler),
		Addr:    "0.0.0.0:8080",
	}
	log.Fatal(srv.ListenAndServe())
}
