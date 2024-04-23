package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"github.com/pressly/goose"
	"golang.org/x/crypto/bcrypt"
)

const (
	dbConnectionString = "user=postgres dbname=postgres password=admin sslmode=disable"
	migrationsDir      = "./migrations"
)

var (
	db    *sql.DB
	store = sessions.NewCookieStore([]byte("your-very-secure-key"))
)

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
	r.HandleFunc("/delete-comment", deleteCommentHandler).Methods("POST")
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/comments", commentsHandler).Methods("GET")
	r.HandleFunc("/comments", addCommentHandler).Methods("POST")
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Login/Register</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			background-color: #f4f4f4;
			margin: 0;
		}
		.container {
			width: 80%;
			margin: 0 auto;
			padding-top: 20px;
		}
		form {
			background: white;
			padding: 20px;
			border-radius: 8px;
			box-shadow: 0 0 10px rgba(0,0,0,0.1);
			margin-bottom: 20px;
		}
		label, input, textarea {
			display: block;
			width: 100%;
			margin-bottom: 10px;
		}
		button {
			padding: 10px 20px;
			background-color: #007BFF;
			color: white;
			border: none;
			border-radius: 5px;
			cursor: pointer;
		}
		button:hover {
			background-color: #0056b3;
		}
		#comments {
			margin-top: 20px;
		}
		.comment {
			background: white;
			padding: 10px;
			margin-bottom: 10px;
			border-radius: 5px;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Authorization</h1>
		<form id="authForm">
			<label for="username">Username:</label>
			<input type="text" id="username" name="username" required>
			
			<label for="password">Password:</label>
			<input type="password" id="password" name="password" required>
			
			<button type="button" onclick="register()">Register</button>
			<button type="button" onclick="login()">Login</button>
		</form>
		<h2>Comments</h2>
		<form id="commentForm">
			<textarea id="comment" name="comment" required placeholder="Write a comment..."></textarea>
			<button type="button" onclick="addComment()">Post Comment</button>
		</form>
		<div id="comments"></div>
	</div>

	<script>
		function postData(url, data) {
			return fetch(url, {
				method: 'POST',
				headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
				body: data
			})
			.then(response => response.json());
		}

		function getData(url) {
			fetch(url)
				.then(response => response.json())
				.then(data => {
					const commentsDiv = document.getElementById('comments');
					commentsDiv.innerHTML = '';
					data.forEach(comment => {
						const commentDiv = document.createElement('div');
						commentDiv.className = 'comment';
						const formattedDate = new Date(comment.created_at).toLocaleString('en-US', {
							hour12: false,
							year: 'numeric',
							month: '2-digit',
							day: '2-digit',
							hour: '2-digit',
							minute: '2-digit',
							second: '2-digit'
						}).replace(',', '').replace(/(\d+)\/(\d+)\/(\d+), (\d+:\d+:\d+)/, "$4/$2/$3"); // Reformat to HH:MM:SS/DD/MM/YYYY
						commentDiv.textContent = 'ID ' + comment.id + ': ' + comment.username + ' - ' + comment.content + ' (posted on: ' + formattedDate + ')';
						commentsDiv.appendChild(commentDiv);
					});
				});
		}
		function register() {
			const formData = new FormData(document.getElementById('authForm'));
			postData('/register', new URLSearchParams(formData))
				.then(data => alert(data.message));
		}

		function login() {
			const formData = new FormData(document.getElementById('authForm'));
			postData('/login', new URLSearchParams(formData))
				.then(data => {
					alert(data.message);
					if (data.message === 'Login successful') {
						console.log('User logged in:', data.user); // Log user info
						getData('/comments'); // Load comments
						if (data.role === 'admin') {
							console.log('Admin access granted.'); // Log if the user is an admin
							showAdminMenu(); // THIS FUCKING THING NOW GETS CALLED FOR SURE
						}
					}
				}).catch(error => {
					console.error('Failed to log in:', error);
					alert('Login failed, check the console!');
				});
		}
		
		function showAdminMenu() {
			const adminDiv = document.createElement('div');
			adminDiv.id = 'adminMenu';
			adminDiv.innerHTML = '<h2>Admin Menu</h2>' +
				'<input type="text" id="deleteCommentId" placeholder="Enter comment ID">' +
				'<button onclick="deleteComment()">Delete Comment</button>';
			document.querySelector('.container').appendChild(adminDiv);
		}
		
		
		function deleteComment() {
			const commentId = document.getElementById('deleteCommentId').value;
			postData('/delete-comment', 'id=' + commentId)
				.then(response => response.json())  // Make sure you parse the JSON response correctly
				.then(data => {
					alert(data.message);  // Now it will show the actual message from the server
					getData('/comments'); // Refresh comments list
				})
				.catch(error => {
					alert('Failed to delete comment: ' + error);  // Handle any errors that occur during the fetch operation
				});
		}
		

		function addComment() {
			const formData = new FormData(document.getElementById('commentForm'));
			postData('/comments', new URLSearchParams(formData))
				.then(data => {
					alert(data.message);
					document.getElementById('comment').value = ''; // Clear the textarea
					getData('/comments'); // Refresh comments
				});
		}

		window.onload = function() {
			getData('/comments'); // Load comments initially when the page loads
		}
	</script>
</body>
</html>`

	fmt.Fprint(w, html)
}
func addCommentHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		respondJSON(w, http.StatusUnauthorized, "Login required")
		return
	}

	username := session.Values["username"].(string)
	userID, err := getUserIDByUsername(username) // Implement this function based on your user DB schema
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, "Failed to get user ID")
		return
	}

	content := r.FormValue("comment")

	if content == "" {
		respondJSON(w, http.StatusBadRequest, "Comment cannot be empty")
		return
	}
	result, err := db.Exec("INSERT INTO comments (user_id, content) VALUES ($1, $2)", userID, content)
	if err != nil {
		log.Printf("Error inserting comment: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Inserting comment failed: "+err.Error())
		return
	}
	_ = result // Ignore the result if not needed
	respondJSON(w, http.StatusOK, "Comment added successfully")
}

func deleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentID := r.FormValue("id")
	if commentID == "" {
		respondJSON(w, http.StatusBadRequest, "Comment ID required")
		return
	}

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = $1)", commentID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if comment exists: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Error checking comment: "+err.Error())
		return
	}

	if !exists {
		respondJSON(w, http.StatusNotFound, "Comment does not exist")
		return
	}

	if _, err := db.Exec("DELETE FROM comments WHERE id = $1", commentID); err != nil {
		log.Printf("Error deleting comment: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Deleting comment failed: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, "Comment deleted successfully")
}

type Comment struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func getUserIDByUsername(username string) (int, error) {
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE user_name = $1", username).Scan(&userID)
	if err != nil {
		log.Printf("Couldn't find the damn user: %v", err)
		return 0, err // Return zero and the error if something goes wrong
	}
	return userID, nil // Return the user ID and no error if all is well
}

func commentsHandler(w http.ResponseWriter, r *http.Request) {
	var comments []Comment
	query := `
        SELECT c.id, u.user_name, c.content, c.created_at 
        FROM comments c
        JOIN users u ON c.user_id = u.id
        ORDER BY c.created_at DESC
    `
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying comments: %v", err)
		respondJSON(w, http.StatusInternalServerError, "Query failed: "+err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.ID, &comment.Username, &comment.Content, &comment.CreatedAt); err != nil {
			log.Printf("Error scanning comments: %v", err)
			continue
		}
		log.Printf("Fetched comment by: %s", comment.Username) // Log to check what usernames are being fetched
		comments = append(comments, comment)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comments)
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

	var hashedPassword, role string
	err := db.QueryRow("SELECT password_hash, role FROM users WHERE user_name = $1", username).Scan(&hashedPassword, &role)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, "User not found or query failed")
		return
	}

	if CheckPasswordHash(password, hashedPassword) {
		session, _ := store.Get(r, "session-name")
		session.Values["authenticated"] = true
		session.Values["username"] = username
		session.Values["role"] = role // Make sure to save the role in the session
		session.Save(r, w)

		response := map[string]string{
			"message": "Login successful",
			"role":    role,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshaling JSON response: %v", err)
			respondJSON(w, http.StatusInternalServerError, "Failed to create JSON response")
			return
		}
		respondJSON(w, http.StatusOK, string(jsonResponse))
	} else {
		respondJSON(w, http.StatusUnauthorized, "Invalid password")
	}
}

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
