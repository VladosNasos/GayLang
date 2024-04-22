package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/pressly/goose"
	"golang.org/x/crypto/bcrypt"
)

const (
	dbConnectionString = "user=postgres dbname=postgres password=admin sslmode=disable"
	migrationsDir      = "./migrations"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler).Methods("POST")

	// r.HandleFunc("/comments", commentsHandler).Methods("GET")
	// r.HandleFunc("/comments", addCommentHandler).Methods("POST")
	// r.HandleFunc("/delete-comment", deleteCommentHandler).Methods("POST")
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
    <!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login/Register</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            padding: 20px;
            background-color: #f4f4f4;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
        }
        form {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
            width: 300px; /* Set the width of the form */
        }
        label, input {
            margin-bottom: 10px;
            display: block;
        }
        input[type="text"], input[type="password"] {
            width: 100%; /* Full width of the form */
            box-sizing: border-box; /* Include padding in width calculation */
            padding: 10px;
        }
        button {
            padding: 10px 20px;
            margin-top: 10px;
            background-color: #007BFF;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            width: 100%; /* Full width of the form */
        }
        button:hover {
            background-color: #0056b3;
        }
    </style>
</head>
<body>
    <h1 style="position: absolute; top: 20px; width: 100%; text-align: center;">Authorization</h1>
    <form id="form">
        <label for="username">Username:</label>
        <input type="text" id="username" name="username" required>
        
        <label for="password">Password:</label>
        <input type="password" id="password" name="password" required>
        
        <button type="button" onclick="register()">Register</button>
        <button type="button" onclick="login()">Login</button>
    </form>

    <script>
        function postData(url, data) {
            return fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: data
            })
            .then(response => response.json());
        }

        function register() {
            const formData = new FormData(document.getElementById('form'));
            postData('/register', new URLSearchParams(formData))
                .then(data => alert(data.message));
        }

        function login() {
            const formData = new FormData(document.getElementById('form'));
            postData('/login', new URLSearchParams(formData))
                .then(data => alert(data.message));
        }
    </script>
</body>
</html>

    `
	fmt.Fprint(w, html)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	// Initial logging to confirm handler entry
	log.Println("Entered registerHandler")

	username := r.FormValue("username")
	password := r.FormValue("password")
	log.Printf("Registering username: %s", username) // Log the username attempting to register

	if username == "" || password == "" {
		respondJSON(w, http.StatusBadRequest, "Username and password required")
		return
	}

	var userExists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE user_name = $1", username).Scan(&userExists)
	if err != nil {
		log.Printf("Error querying user existence: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Query failed: "+err.Error())
		return
	}

	if userExists > 0 {
		respondJSON(w, http.StatusBadRequest, "Username already exists")
		return
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Password hashing failed")
		return
	}

	_, err = db.Exec("INSERT INTO users (user_name, password_hash) VALUES ($1, $2)", username, hashedPassword)
	if err != nil {
		log.Printf("Error inserting new user: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Inserting user failed")
		return
	}

	respondJSON(w, http.StatusOK, "Registration successful")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		respondJSON(w, http.StatusBadRequest, "Username and password required")
		return
	}

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, "Database connection failed")
		return
	}
	defer db.Close()

	var hashedPassword string
	err = db.QueryRow("SELECT password_hash FROM users WHERE user_name = $1", username).Scan(&hashedPassword)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, "User not found or query failed")
		return
	}

	if !CheckPasswordHash(password, hashedPassword) {
		respondJSON(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	respondJSON(w, http.StatusOK, "Login successful")

}

// func commentsHandler(w http.ResponseWriter, r *http.Request) {
// 	rows, err := db.Query("SELECT id, content, user_name FROM comments JOIN users ON comments.user_id = users.id")
// 	if err != nil {
// 		respondJSON(w, http.StatusInternalServerError, "Failed to query comments")
// 		return
// 	}
// 	defer rows.Close()

// 	comments := []map[string]interface{}{}
// 	for rows.Next() {
// 		var id int
// 		var content, userName string
// 		if err := rows.Scan(&id, &content, &userName); err != nil {
// 			respondJSON(w, http.StatusInternalServerError, "Failed to scan comments")
// 			return
// 		}
// 		comments = append(comments, map[string]interface{}{
// 			"id":       id,
// 			"content":  content,
// 			"userName": userName,
// 		})
// 	}

// 	respondJSON(w, http.StatusOK, comments)
// }
// func addCommentHandler(w http.ResponseWriter, r *http.Request) {
// 	userID := getUserIDFromContext(r) // You need to implement a way to get user ID from the request context/session
// 	content := r.FormValue("content")

// 	if userID == 0 || content == "" {
// 		respondJSON(w, http.StatusBadRequest, "Invalid user or empty content")
// 		return
// 	}

// 	_, err := db.Exec("INSERT INTO comments (user_id, content) VALUES ($1, $2)", userID, content)
// 	if err != nil {
// 		respondJSON(w, http.StatusInternalServerError, "Failed to insert comment")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, "Comment added")
// }
// func deleteCommentHandler(w http.ResponseWriter, r *http.Request) {
// 	adminPassword := r.FormValue("adminPassword")
// 	commentID := r.FormValue("commentID")

// 	if !isAdmin(adminPassword) { // You need to implement isAdmin function
// 		respondJSON(w, http.StatusUnauthorized, "Unauthorized access")
// 		return
// 	}

// 	_, err := db.Exec("DELETE FROM comments WHERE id = $1", commentID)
// 	if err != nil {
// 		respondJSON(w, http.StatusInternalServerError, "Failed to delete comment")
// 		return
// 	}

//		respondJSON(w, http.StatusOK, "Comment deleted")
//	}
func respondJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
