package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/golang-jwt/jwt/v5"
)

type RequestPayload struct {
	Message  string `json:"message"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`  // Optional field for redirect URL
	Token   string `json:"token,omitempty"` // Add this field
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
	Email             string
	Password          string
	Role              string
	Active            bool
	ConfirmationToken string    // New field for email confirmation
	Name              string    // Add this field
	Avatar            string    // Add this field
	OTP               string    // Add this field
	OTPExpiry         time.Time // Add this field
}

type Booking struct {
	gorm.Model
	Date  string
	Time  string
	Field string
}

var db *gorm.DB

// Add JWT secret key
var jwtSecret = []byte(os.Getenv("asdasd"))

// Add JWT claims struct
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

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

// Add middleware to verify JWT token
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "No token provided",
			})
			return
		}

		// Remove "Bearer " prefix if present
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid token",
			})
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// Handle authentication and redirection based on role
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload RequestPayload
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&payload)
		defer r.Body.Close()
		if err != nil || payload.Email == "" || payload.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Invalid credentials",
			})
			return
		}

		var user User
		db.Where("email = ?", payload.Email).First(&user)
		if user.ID == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "User not found",
			})
			return
		}

		if user.Password != payload.Password {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Incorrect password",
			})
			return
		}

		if !user.Active {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Please confirm your email address before logging in",
			})
			return
		}

		// Generate and save OTP
		otp := generateOTP()
		user.OTP = otp
		user.OTPExpiry = time.Now().Add(5 * time.Minute) // OTP valid for 5 minutes
		db.Save(&user)

		// Send OTP via email
		emailBody := fmt.Sprintf("Your OTP for login is: %s\nThis OTP will expire in 5 minutes.", otp)
		err = sendEmailWithAttachment(
			user.Email,
			"Your Login OTP",
			emailBody,
			"",
			"",
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Failed to send OTP",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "pending_otp",
			Message: "OTP has been sent to your email",
		})
		return
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

		// Check if user already exists
		var existingUser User
		if result := db.Where("email = ?", user.Email).First(&existingUser); result.Error == nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Email already registered",
			})
			return
		}

		// Generate confirmation token
		confirmationToken := generateConfirmationToken()
		user.ConfirmationToken = confirmationToken
		user.Role = "user"
		user.Active = false // User starts as inactive

		// Create confirmation link
		confirmationLink := fmt.Sprintf("http://localhost:8080/confirm?token=%s", confirmationToken)

		// Send confirmation email
		emailBody := fmt.Sprintf("Please confirm your email by clicking this link: %s", confirmationLink)
		err = sendEmailWithAttachment(
			user.Email,
			"Confirm your SportLife registration",
			emailBody,
			"",
			"",
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Failed to send confirmation email",
			})
			return
		}

		db.Create(&user)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Registration successful. Please check your email to confirm your account.",
		})
		return
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
	username := "" // Enter your email here
	password := "" // Enter your smtp password here
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

// Add this function after initDB()
func generateConfirmationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Add new confirmation handler
func handleConfirmEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	var user User
	result := db.Where("confirmation_token = ?", token).First(&user)
	if result.Error != nil {
		http.Error(w, "Invalid or expired token", http.StatusBadRequest)
		return
	}

	// Activate user
	user.Active = true
	user.ConfirmationToken = "" // Clear the token
	db.Save(&user)

	// Redirect to login page with success message
	http.Redirect(w, r, "http://localhost:5500/assignment1/front/form.html?confirmed=true", http.StatusTemporaryRedirect)
}

// Add new handler for profile updates
func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := r.Context().Value("claims").(*Claims)
	var user User

	if err := db.Where("email = ?", claims.Email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Update name if provided
	if name := r.FormValue("name"); name != "" {
		user.Name = name
	}

	// Handle avatar upload
	file, handler, err := r.FormFile("avatar")
	if err == nil {
		defer file.Close()

		// Generate unique filename
		filename := fmt.Sprintf("avatars/%d_%s", user.ID, handler.Filename)

		// Ensure avatars directory exists
		os.MkdirAll("avatars", 0755)

		// Create new file
		dst, err := os.Create(filename)
		if err != nil {
			http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy uploaded file to destination
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
			return
		}

		user.Avatar = filename
	}

	// Save updates
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "Profile updated successfully",
	})
}

// Add new handler for fetching profile
func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := r.Context().Value("claims").(*Claims)
	var user User

	if err := db.Where("email = ?", claims.Email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return user profile data
	json.NewEncoder(w).Encode(map[string]string{
		"email":  user.Email,
		"name":   user.Name,
		"avatar": user.Avatar,
	})
}

// Add this function to generate OTP
func generateOTP() string {
	// Generate 6-digit OTP
	rand.Seed(time.Now().UnixNano())
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	return otp
}

// Add new handler for OTP verification
func handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.OTP != payload.OTP || time.Now().After(user.OTPExpiry) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Invalid or expired OTP",
		})
		return
	}

	// Clear OTP after successful verification
	user.OTP = ""
	user.OTPExpiry = time.Time{}
	db.Save(&user)

	// Generate JWT token and return success response
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Could not generate token",
		})
		return
	}

	// Determine redirect URL based on role
	redirectURL := ""
	switch user.Role {
	case "admin":
		redirectURL = "admin-panel.html"
	case "user":
		redirectURL = "user-profile.html"
	default:
		redirectURL = "home.html"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "Login successful",
		Data:    redirectURL,
		Token:   tokenString,
	})
}

// Add these new handler functions

func handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userId := urlParts[3]

	// Parse request body
	var payload struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate role
	if payload.Role != "user" && payload.Role != "admin" {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Update user role in database
	var user User
	if err := db.First(&user, userId).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Role = payload.Role
	if err := db.Save(&user).Error; err != nil {
		http.Error(w, "Failed to update user role", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "User role updated successfully",
	})
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userId := urlParts[3]

	// Delete user from database
	if err := db.Delete(&User{}, userId).Error; err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "User deleted successfully",
	})
}

func handleDeleteBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract booking ID from URL
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	bookingId := urlParts[3]

	// Delete booking from database
	if err := db.Delete(&Booking{}, bookingId).Error; err != nil {
		http.Error(w, "Failed to delete booking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResponsePayload{
		Status:  "success",
		Message: "Booking deleted successfully",
	})
}

func main() {
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handleAuth)
	// Protect these endpoints with JWT middleware
	mux.HandleFunc("/bookings", authMiddleware(handleBookings))
	mux.HandleFunc("/admin/users", authMiddleware(handleGetUsers))
	mux.HandleFunc("/admin/bookings", authMiddleware(handleGetBookings))
	mux.HandleFunc("/admin/email", authMiddleware(handleSendEmail))
	mux.HandleFunc("/confirm", handleConfirmEmail)
	mux.HandleFunc("/profile/update", authMiddleware(handleUpdateProfile))
	mux.HandleFunc("/profile", authMiddleware(handleGetProfile))
	mux.HandleFunc("/verify-otp", handleVerifyOTP)

	// Add new admin routes
	mux.HandleFunc("/admin/users/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/role") {
			handleUpdateUserRole(w, r)
		} else {
			handleDeleteUser(w, r)
		}
	}))
	mux.HandleFunc("/admin/bookings/", authMiddleware(handleDeleteBooking))

	fmt.Println("Server started on port 8080...")
	if err := http.ListenAndServe(":8080", enableCORS(mux)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
