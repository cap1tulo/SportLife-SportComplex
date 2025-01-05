package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type RequestPayload struct {
	Message string `json:"message"`
}

type ResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
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
	dsn := "user=postgres password=LUFFYtaroo111&&& dbname=SportLife sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}

	// Миграция моделей User и Booking
	db.AutoMigrate(&User{}, &Booking{})
	fmt.Println("Успешно подключено к базе данных и выполнена миграция")
}
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Разрешить запросы с любых доменов
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Обработка preflight-запросов
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Авторизация и регистрация
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Обработка авторизации
		var user User
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		defer r.Body.Close()
		if err != nil || user.Email == "" || user.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Некорректные данные для авторизации",
			})
			return
		}

		var existingUser User
		db.Where("email = ?", user.Email).First(&existingUser)
		if existingUser.ID == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Пользователь не найден",
			})
			return
		}

		if existingUser.Password != user.Password {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Неверный пароль",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Успешный вход",
		})
	} else if r.Method == http.MethodPut {
		// Регистрация нового пользователя
		var user User
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&user)
		defer r.Body.Close()
		if err != nil || user.Email == "" || user.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Некорректные данные для регистрации",
			})
			return
		}

		db.Create(&user)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Пользователь успешно зарегистрирован",
		})
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Метод не поддерживается",
		})
	}
}

// CRUD операции для бронирования
func handleBookings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Создание нового бронирования
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.Date == "" || booking.Time == "" || booking.Field == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Некорректные данные бронирования",
			})
			return
		}

		db.Create(&booking)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Бронирование создано",
		})
	} else if r.Method == http.MethodGet {
		// Получение всех бронирований
		var bookings []Booking
		db.Find(&bookings)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(bookings)
	} else if r.Method == http.MethodPut {
		// Обновление бронирования
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Некорректные данные бронирования",
			})
			return
		}

		db.Save(&booking)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Бронирование обновлено",
		})
	} else if r.Method == http.MethodDelete {
		// Удаление бронирования
		var booking Booking
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&booking)
		defer r.Body.Close()
		if err != nil || booking.ID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponsePayload{
				Status:  "fail",
				Message: "Некорректные данные бронирования",
			})
			return
		}

		db.Delete(&booking)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "success",
			Message: "Бронирование удалено",
		})
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ResponsePayload{
			Status:  "fail",
			Message: "Метод не поддерживается",
		})
	}
}
func main() {
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handleAuth)
	mux.HandleFunc("/bookings", handleBookings)

	fmt.Println("Сервер запущен на порту 8080...")
	if err := http.ListenAndServe(":8080", enableCORS(mux)); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
