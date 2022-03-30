// Package sqlitedb sets up the database, and handles all interactions with it
package store

import (
	"database/sql"
	"log"
	"os"

	"go-server/structs"

	"github.com/blockloop/scan"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB
var dbPath = "./todo.db"

type Todo = structs.Todo

//Create the todos table
func createTable(db *sql.DB) {
	createTodoTableSql := `CREATE TABLE IF NOT EXISTS todos (
    ID TEXT PRIMARY KEY,
    Title TEXT NOT NULL,
    Completed BOOLEAN NOT NULL,
    UserEmail TEXT NOT NULL,
    UserSub TEXT NOT NULL
	);`

	log.Println("Create todos table...")

	statement, err := db.Prepare(createTodoTableSql) // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec() // Execute SQL Statements
	log.Println("todos table created")
}

//Initialize the database
func InitDB() {
	log.Println("Creating todo.db...")
	if _, fileExistsError := os.Stat(dbPath); os.IsNotExist(fileExistsError){
		file, err := os.Create(dbPath)
			if err != nil {
			log.Fatal(err.Error())
		}
		file.Close()
		log.Println("todo.db created")
	}

	sqliteDatabase, _ := sql.Open("sqlite3", dbPath) // Open the created SQLite File

	createTable(sqliteDatabase)
	DB = sqliteDatabase
}

//Get all todos
func GetTodos() ([]Todo, error) {
	var todos []Todo

	rows, err := DB.Query("SELECT * FROM todos")

	if err != nil {
		return nil, err
	} else {
		scanErr := scan.Rows(&todos, rows)
		if (scanErr != nil) {
			return nil, scanErr
		} else {
			return todos, nil
		}
	}
}

//Insert a todo
func InsertTodo(todo Todo) error {
	_, err := DB.Exec(`INSERT INTO todos (ID, UserEmail, Title, Completed, UserSub) VALUES (?, ?, ?, ?, ?)`, todo.ID, todo.UserEmail, todo.Title, todo.Completed, todo.UserSub)

	if err != nil {
		return err
	} else {
		return nil
	}
}

//Update a todo
func UpdateTodo(todo Todo) error {
	_, err := DB.Exec(`UPDATE todos SET UserEmail=?, Title=?, UserSub=?, Completed=? WHERE ID=?`, todo.UserEmail, todo.Title, todo.UserSub, todo.Completed, todo.ID)

	if err != nil {
		return err
	} else {
		return nil
	}
}

//Delete a todo
func DeleteTodo(todo Todo) error {
	_, err := DB.Exec(`DELETE FROM todos WHERE ID=?`, todo.ID)

	if err != nil {
		return err
	} else {
		return nil
	}
}
