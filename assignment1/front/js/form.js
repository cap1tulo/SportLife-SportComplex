document.getElementById("registerForm").addEventListener("submit", async (e) => {
    e.preventDefault();

    const name = document.getElementById("name").value;
    const email = document.getElementById("email").value;
    const phone = document.getElementById("phone").value;

    const response = await fetch("http://localhost:8080/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, phone })
    });

    const data = await response.json();
    const responseMessage = document.getElementById("responseMessage");

    if (response.ok) {
        responseMessage.style.color = "green";
        responseMessage.innerText = data.message;
    } else {
        responseMessage.style.color = "red";
        responseMessage.innerText = data.message;
    }
});
