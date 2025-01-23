// Add this at the beginning of the file
document.addEventListener('DOMContentLoaded', () => {
    checkAuthStatus();
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('confirmed') === 'true') {
        alert('Email confirmed successfully! You can now log in.');
        // Remove the query parameter
        window.history.replaceState({}, '', window.location.pathname);
    }
});

function checkAuthStatus() {
    const token = localStorage.getItem('token');
    const greetingDiv = document.getElementById('greeting');
    const loginContainer = document.getElementById('login-container');
    
    if (token) {
        // Decode the JWT token to get the email
        const payload = JSON.parse(atob(token.split('.')[1]));
        document.getElementById('user-email').textContent = payload.email;
        greetingDiv.style.display = 'block';
        loginContainer.style.display = 'none';
    } else {
        greetingDiv.style.display = 'none';
        loginContainer.style.display = 'block';
    }
}

// Add logout button handler
document.getElementById('logout-button')?.addEventListener('click', () => {
    logout();
    checkAuthStatus();
});

// Login form submission
document.getElementById("login-form").addEventListener("submit", async (event) => {
    event.preventDefault();

    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;

    try {
        const response = await fetch("http://localhost:8080/auth", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password }),
        });

        const data = await response.json();

        if (response.ok) {
            if (data.status === "pending_otp") {
                // Show OTP input field
                showOTPInput(email);
                alert(data.message);
            } else if (data.status === "success") {
                localStorage.setItem('token', data.token);
                checkAuthStatus();
                alert(data.message);
                window.location.href = data.data;
            }
        } else {
            alert(data.message || "Login failed. Please try again.");
        }
    } catch (error) {
        console.error("Error during login:", error);
        alert("An error occurred. Please try again later.");
    }
});

function showOTPInput(email) {
    const loginForm = document.getElementById("login-form");
    loginForm.innerHTML = `
        <input type="text" id="otp" placeholder="Enter OTP" required>
        <button type="button" onclick="verifyOTP('${email}')">Verify OTP</button>
    `;
}

async function verifyOTP(email) {
    const otp = document.getElementById("otp").value;
    
    try {
        const response = await fetch("http://localhost:8080/verify-otp", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, otp }),
        });

        const data = await response.json();

        if (response.ok && data.status === "success") {
            localStorage.setItem('token', data.token);
            checkAuthStatus();
            alert(data.message);
            window.location.href = data.data;
        } else {
            alert(data.message || "OTP verification failed. Please try again.");
        }
    } catch (error) {
        console.error("Error during OTP verification:", error);
        alert("An error occurred. Please try again later.");
    }
}

// Registration button click
document.getElementById("register-button").addEventListener("click", async () => {
    const email = document.getElementById("email").value;
    const password = document.getElementById("password").value;

    try {
        const response = await fetch("http://localhost:8080/auth", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password }),
        });

        const data = await response.json();
        alert(data.message);
        
        if (data.status === "success") {
            // Clear the form
            document.getElementById("email").value = "";
            document.getElementById("password").value = "";
        }
    } catch (error) {
        console.error("Error during registration:", error);
        alert("An error occurred. Please try again later.");
    }
});

// Add a function to check if user is authenticated
function isAuthenticated() {
    return localStorage.getItem('token') !== null;
}

// Add a function to get the JWT token
function getToken() {
    return localStorage.getItem('token');
}

// Update the logout function
function logout() {
    localStorage.removeItem('token');
    checkAuthStatus();
}

// Example of how to make authenticated requests
async function makeAuthenticatedRequest(url, method = 'GET', body = null) {
    const token = getToken();
    const headers = {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
    };

    const options = {
        method,
        headers,
        body: body ? JSON.stringify(body) : null,
    };

    const response = await fetch(url, options);
    
    if (response.status === 401) {
        // Token is invalid or expired
        logout();
        return null;
    }

    return response;
}

