package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	// Подключаем CORS
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var db *sql.DB

// Структура для данных пользователя
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Структура для данных бронирования
type Booking struct {
	UserName  string `json:"user_name"`  // Имя пользователя
	FieldName string `json:"field_name"` // Название поля
	Date      string `json:"date"`       // Дата бронирования
	TimeSlot  string `json:"time_slot"`  // Временной слот
}

// Подключение к базе данных
func initDB() {
	var err error
	connStr := "user=postgres password=LUFFYtaroo111&&& dbname=SportLife sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}
	fmt.Println("Подключение к базе данных успешно!")
}

// Обработчик для регистрации пользователя
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"status":"fail", "message":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	// Проверка обязательных полей
	if user.Name == "" || user.Email == "" || user.Phone == "" {
		http.Error(w, `{"status":"fail", "message":"Отсутствуют обязательные поля"}`, http.StatusBadRequest)
		return
	}

	// Сохранение данных в базу
	_, err := db.Exec("INSERT INTO users (name, email, phone) VALUES ($1, $2, $3)", user.Name, user.Email, user.Phone)
	if err != nil {
		http.Error(w, `{"status":"fail", "message":"Ошибка сохранения данных"}`, http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Пользователь успешно зарегистрирован"})
}

// Обработчик для бронирования
func bookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"status":"fail", "message":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	var booking Booking
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		http.Error(w, `{"status":"fail", "message":"Некорректный JSON"}`, http.StatusBadRequest)
		return
	}

	if booking.UserName == "" || booking.FieldName == "" || booking.Date == "" || booking.TimeSlot == "" {
		http.Error(w, `{"status":"fail", "message":"Отсутствуют обязательные поля"}`, http.StatusBadRequest)
		return
	}

	if booking.FieldName != "Поле Бекет Батыра" && booking.FieldName != "Поле Орынбаева" {
		http.Error(w, `{"status":"fail", "message":"Некорректное название поля"}`, http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE name = $1", booking.UserName).Scan(&userID)
	if err != nil {
		http.Error(w, `{"status":"fail", "message":"Пользователь не найден"}`, http.StatusNotFound)
		return
	}

	_, err = db.Exec("INSERT INTO bookings (user_id, field_name, date, time_slot) VALUES ($1, $2, $3, $4)",
		userID, booking.FieldName, booking.Date, booking.TimeSlot)
	if err != nil {
		http.Error(w, `{"status":"fail", "message":"Ошибка сохранения данных"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Бронирование успешно создано"})
}

func main() {
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/booking", bookingHandler)

	handler := cors.AllowAll().Handler(mux)

	fmt.Println("Сервер запущен на порту 8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
