document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('token');
    if (!token) {
        window.location.href = 'form.html';
        return;
    }

    // Verify if user is admin
    const payload = JSON.parse(atob(token.split('.')[1]));
    if (payload.role !== 'admin') {
        window.location.href = 'form.html';
        return;
    }

    // Fetch and display data
    fetchUsers();
    fetchBookings();

    // Setup logout handler
    document.getElementById('logout-button').addEventListener('click', () => {
        localStorage.removeItem('token');
        window.location.href = 'form.html';
    });
});

async function fetchUsers() {
    try {
        const response = await fetch('http://localhost:8080/admin/users', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to fetch users');
        }

        const users = await response.json();
        displayUsers(users);
    } catch (error) {
        showAlert('Error fetching users: ' + error.message, 'danger');
    }
}

async function fetchBookings() {
    try {
        const response = await fetch('http://localhost:8080/admin/bookings', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to fetch bookings');
        }

        const bookings = await response.json();
        displayBookings(bookings);
    } catch (error) {
        showAlert('Error fetching bookings: ' + error.message, 'danger');
    }
}

function displayUsers(users) {
    const tbody = document.getElementById('users-table-body');
    tbody.innerHTML = users.map(user => `
        <tr>
            <td>${user.ID}</td>
            <td>${user.Email}</td>
            <td>
                <select class="form-select role-select" data-user-id="${user.ID}" 
                        onchange="updateUserRole(${user.ID}, this.value)">
                    <option value="user" ${user.Role === 'user' ? 'selected' : ''}>User</option>
                    <option value="admin" ${user.Role === 'admin' ? 'selected' : ''}>Admin</option>
                </select>
            </td>
            <td>${user.Active ? '<span class="badge bg-success">Active</span>' : 
                              '<span class="badge bg-warning">Pending</span>'}</td>
            <td>${new Date(user.CreatedAt).toLocaleString()}</td>
            <td>
                <button class="btn btn-danger btn-sm" onclick="deleteUser(${user.ID})">
                    Delete
                </button>
            </td>
        </tr>
    `).join('');
}

function displayBookings(bookings) {
    const tbody = document.getElementById('bookings-table-body');
    tbody.innerHTML = bookings.map(booking => `
        <tr>
            <td>${booking.ID}</td>
            <td>${booking.Date}</td>
            <td>${booking.Time}</td>
            <td>${booking.Field}</td>
            <td>${new Date(booking.CreatedAt).toLocaleString()}</td>
            <td>
                <button class="btn btn-danger btn-sm" onclick="deleteBooking(${booking.ID})">
                    Delete
                </button>
            </td>
        </tr>
    `).join('');
}

async function updateUserRole(userId, newRole) {
    try {
        const response = await fetch(`http://localhost:8080/admin/users/${userId}/role`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            },
            body: JSON.stringify({ role: newRole })
        });

        if (!response.ok) {
            throw new Error('Failed to update user role');
        }

        showAlert('User role updated successfully', 'success');
        await fetchUsers(); // Refresh the users list
    } catch (error) {
        showAlert('Error updating user role: ' + error.message, 'danger');
    }
}

async function deleteUser(userId) {
    if (!confirm('Are you sure you want to delete this user?')) {
        return;
    }

    try {
        const response = await fetch(`http://localhost:8080/admin/users/${userId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to delete user');
        }

        showAlert('User deleted successfully', 'success');
        await fetchUsers(); // Refresh the users list
    } catch (error) {
        showAlert('Error deleting user: ' + error.message, 'danger');
    }
}

async function deleteBooking(bookingId) {
    if (!confirm('Are you sure you want to delete this booking?')) {
        return;
    }

    try {
        const response = await fetch(`http://localhost:8080/admin/bookings/${bookingId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            }
        });

        if (!response.ok) {
            throw new Error('Failed to delete booking');
        }

        showAlert('Booking deleted successfully', 'success');
        await fetchBookings(); // Refresh the bookings list
    } catch (error) {
        showAlert('Error deleting booking: ' + error.message, 'danger');
    }
}

function showAlert(message, type) {
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
    alertDiv.role = 'alert';
    alertDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;
    document.querySelector('.container').insertBefore(alertDiv, document.querySelector('.card'));
    
    // Auto dismiss after 3 seconds
    setTimeout(() => {
        alertDiv.remove();
    }, 3000);
}

// Add these styles to your existing style.css file
