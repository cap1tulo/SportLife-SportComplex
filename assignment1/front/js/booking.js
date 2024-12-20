document.getElementById("bookingForm").addEventListener("submit", async (e) => {
	e.preventDefault();

	const userName = document.getElementById("user_name").value;
	const fieldName = document.getElementById("field_name").value;
	const date = document.getElementById("date").value;
	const timeSlot = document.getElementById("time_slot").value;

	try {
		const response = await fetch("http://localhost:8080/booking", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ user_name: userName, field_name: fieldName, date, time_slot: timeSlot }),
		});

		const data = await response.json();
		const responseMessage = document.getElementById("bookingResponse");

		if (response.ok) {
			responseMessage.style.color = "green";
			responseMessage.innerText = data.message;
		} else {
			responseMessage.style.color = "red";
			responseMessage.innerText = data.message;
		}
	} catch (err) {
		console.error("Ошибка запроса:", err);
		const responseMessage = document.getElementById("bookingResponse");
		responseMessage.style.color = "red";
		responseMessage.innerText = "Ошибка отправки запроса!";
	}
});
