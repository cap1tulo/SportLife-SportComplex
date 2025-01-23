document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = 'form.html';
        return;
    }

    // Display user info from token
    const payload = JSON.parse(atob(token.split('.')[1]));
    document.getElementById('user-email').textContent = payload.email;
    document.getElementById('user-token').textContent = token;

    // Handle password change
    const passwordForm = document.getElementById('password-form');
    passwordForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const newPassword = document.getElementById('new-password').value;

        try {
            const response = await fetch('http://localhost:8080/auth', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': token
                },
                body: JSON.stringify({
                    email: payload.email,
                    password: newPassword
                })
            });

            const data = await response.json();
            if (response.ok) {
                showAlert('Password updated successfully!', 'success');
                passwordForm.reset();
            } else {
                showAlert(data.message || 'Failed to update password', 'danger');
            }
        } catch (error) {
            showAlert('Error updating password', 'danger');
        }
    });

    // Handle logout
    document.getElementById('logout-button').addEventListener('click', () => {
        localStorage.removeItem('token');
        window.location.href = 'form.html';
    });
});

function showAlert(message, type) {
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
    alertDiv.role = 'alert';
    alertDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;
    document.querySelector('.container').insertBefore(alertDiv, document.querySelector('.row'));
    
    // Auto dismiss after 3 seconds
    setTimeout(() => {
        alertDiv.remove();
    }, 3000);
}
