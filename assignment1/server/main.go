package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/smtp"
)

type RequestPayload struct {
	Message string `json:"message"`
}

type ResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type EmailPayload struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	File     string `json:"file"`     // Base64 encoded file content
	FileName string `json:"fileName"` // File name for the attachment
}

type User struct {
	gorm.Model
	Email    string
	Password string
}

type Booking struct {
	gorm.Model
	Date  string
	Time  string
	Field string
}

var db *gorm.DB

func initDB() {
	dsn := "user=postgres password=asdasd123123asdasd dbname=sportlife sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Migrate models
	db.AutoMigrate(&User{}, &Booking{})
	fmt.Println("Database connected and migrations applied.")
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Authentication and registration
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var user User
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		defer r.Body.Close()
		if err != nil || user.Email == "" || user.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid credentials",
			})
			return
		}

		var existingUser User
		db.Where("email = ?", user.Email).First(&existingUser)
		if existingUser.ID == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "User not found",
			})
			return
		}

		if existingUser.Password != user.Password {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Incorrect password",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Login successful",
		})
	} else if r.Method == http.MethodPut {
		var user User
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		defer r.Body.Close()
		if err != nil || user.Email == "" || user.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid registration data",
			})
			return
		}

		db.Create(&user)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "User registered successfully",
		})
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Method not supported",
		})
	}
}

// Booking CRUD operations
func handleBookings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.Date == "" || booking.Time == "" || booking.Field == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid booking data",
			})
			return
		}

		db.Create(&booking)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Booking created",
		})
	} else if r.Method == http.MethodGet {
		var bookings []Booking
		db.Find(&bookings)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(bookings)
	} else if r.Method == http.MethodPut {
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid booking data",
			})
			return
		}

		db.Save(&booking)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Booking updated",
		})
	} else if r.Method == http.MethodDelete {
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid booking data",
			})
			return
		}

		db.Delete(&booking)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Booking deleted",
		})
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Method not supported",
		})
	}
}

// Fetch all users
func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var users []User
	db.Find(&users)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Fetch all bookings
func handleGetBookings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var bookings []Booking
	db.Find(&bookings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}

// Email sending with attachment support
func sendEmailWithAttachment(to, subject, body, fileContent, fileName string) error {
	smtpHost := "smtp.mail.ru"
	smtpPort := "465"
	username := "ploc91@mail.ru"
	password := ""
	from := username

	// Boundary for separating email parts
	boundary := "my-boundary-12345"

	// Email header
	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n",
		from, to, subject, boundary,
	)

	// Email body
	bodyPart := fmt.Sprintf(
		"--%s\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s\r\n",
		boundary, body,
	)

	// Attachment part (if fileContent is provided)
	attachmentPart := ""
	if fileContent != "" {
		decodedFile, err := base64.StdEncoding.DecodeString(fileContent)
		if err != nil {
			log.Println("Error decoding file content:", err)
			return err
		}

		attachmentPart = fmt.Sprintf(
			"--%s\r\nContent-Type: application/octet-stream\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"%s\"\r\n\r\n%s\r\n",
			boundary, fileName, base64.StdEncoding.EncodeToString(decodedFile),
		)
	}

	// Closing boundary
	closingBoundary := fmt.Sprintf("--%s--", boundary)

	// Final email message
	message := header + bodyPart + attachmentPart + closingBoundary

	// Connect to the SMTP server
	auth := smtp.PlainAuth("", username, password, smtpHost)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpHost,
	}

	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsConfig)
	if err != nil {
		log.Println("TLS connection error:", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Println("SMTP client error:", err)
		return err
	}

	if err := client.Auth(auth); err != nil {
		log.Println("SMTP authentication error:", err)
		return err
	}

	if err := client.Mail(from); err != nil {
		log.Println("Error setting sender email:", err)
		return err
	}

	if err := client.Rcpt(to); err != nil {
		log.Println("Error setting recipient email:", err)
		return err
	}

	writer, err := client.Data()
	if err != nil {
		log.Println("Error getting writer:", err)
		return err
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		log.Println("Error writing email message:", err)
		return err
	}

	if err := writer.Close(); err != nil {
		log.Println("Error closing writer:", err)
		return err
	}

	if err := client.Quit(); err != nil {
		log.Println("Error quitting client:", err)
		return err
	}

	log.Println("Email sent successfully with attachment.")
	return nil
}

// Email handler function
func handleSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var emailPayload EmailPayload
	if err := json.NewDecoder(r.Body).Decode(&emailPayload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Send email
	err := sendEmailWithAttachment(emailPayload.To, emailPayload.Subject, emailPayload.Body, emailPayload.File, emailPayload.FileName)
	if err != nil {
		log.Println("Error sending email:", err)
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "Email sent successfully with attachment",
	})
}

func main() {
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handleAuth)
	mux.HandleFunc("/bookings", handleBookings)
	mux.HandleFunc("/admin/users", handleGetUsers)
	mux.HandleFunc("/admin/bookings", handleGetBookings)
	mux.HandleFunc("/admin/email", handleSendEmail)

	fmt.Println("Server started on port 8080...")
	if err := http.ListenAndServe(":8080", enableCORS(mux)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
