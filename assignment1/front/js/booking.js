// Создание бронирования
document.getElementById("booking-form").addEventListener("submit", async (event) => {
    event.preventDefault();

    // Получение данных из input полей
    const date = document.getElementById("date").value;
    const time = document.getElementById("time").value;
    const field = document.getElementById("field").value;

    // Отправка запроса на сервер
    const response = await fetch("http://localhost:8080/bookings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ date, time, field })
    });

	
    const data = await response.json();
    alert(data.message); // Показываем сообщение пользователю
});

// Получение всех бронирований
document.getElementById("view-bookings").addEventListener("click", async () => {
    // Отправка запроса на сервер
    const response = await fetch("http://localhost:8080/bookings", {
        method: "GET",
    });

    const bookings = await response.json();

    // Отображение списка бронирований
    const bookingsList = document.getElementById("bookings-list");
    bookingsList.innerHTML = ""; // Очистка старого списка

    bookings.forEach((booking) => {
        const item = document.createElement("div");
        item.textContent = `Дата: ${booking.Date}, Время: ${booking.Time}, Поле: ${booking.Field}`;
        bookingsList.appendChild(item);
    });
});

// Обновление бронирования
document.getElementById("update-booking").addEventListener("click", async () => {
    const id = prompt("Введите ID бронирования для обновления:");
    const date = document.getElementById("date").value;
    const time = document.getElementById("time").value;
    const field = document.getElementById("field").value;

    if (id) {
        // Отправка запроса на сервер
        const response = await fetch("http://localhost:8080/bookings", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ ID: parseInt(id), Date: date, Time: time, Field: field })
        });

        const data = await response.json();
        alert(data.message); // Показываем сообщение пользователю
    }
});

// Удаление бронирования
document.getElementById("delete-booking").addEventListener("click", async () => {
    const id = prompt("Введите ID бронирования для удаления:");

    if (id) {
        // Отправка запроса на сервер
        const response = await fetch("http://localhost:8080/bookings", {
            method: "DELETE",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ ID: parseInt(id) })
        });

        const data = await response.json();
        alert(data.message); // Показываем сообщение пользователю
    }
});
