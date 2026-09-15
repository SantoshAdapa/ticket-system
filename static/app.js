document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const authSection = document.getElementById('auth-section');
    const appSection = document.getElementById('app-section');
    const authForm = document.getElementById('auth-form');
    const authTitle = document.getElementById('auth-title');
    const authSubmitBtn = document.getElementById('auth-submit-btn');
    const authToggleLink = document.getElementById('auth-toggle-link');
    const authToggleText = document.getElementById('auth-toggle-text');
    const authError = document.getElementById('auth-error');
    
    const logoutBtn = document.getElementById('logout-btn');
    const createTicketForm = document.getElementById('create-ticket-form');
    const ticketsContainer = document.getElementById('tickets-container');
    const ticketError = document.getElementById('ticket-error');

    // State
    let isLoginMode = true;

    // --- Authentication Flow ---

    // Check if user is already logged in
    function checkAuth() {
        const token = localStorage.getItem('token');
        if (token) {
            authSection.classList.add('hidden');
            appSection.classList.remove('hidden');
            fetchTickets();
        } else {
            authSection.classList.remove('hidden');
            appSection.classList.add('hidden');
        }
    }

    // Toggle between Login and Register
    authToggleLink.addEventListener('click', (e) => {
        e.preventDefault();
        isLoginMode = !isLoginMode;
        authTitle.textContent = isLoginMode ? 'Login to Ticket System' : 'Register New Account';
        authSubmitBtn.textContent = isLoginMode ? 'Login' : 'Register';
        authToggleText.textContent = isLoginMode ? "Don't have an account? " : "Already have an account? ";
        authToggleLink.textContent = isLoginMode ? 'Register here' : 'Login here';
        authError.classList.add('hidden');
        authForm.reset();
    });

    // Handle Auth Submit
    authForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;
        const endpoint = isLoginMode ? '/auth/login' : '/auth/register';

        try {
            const res = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            const data = await res.json();

            if (!res.ok) {
                throw new Error(data.error || 'Authentication failed');
            }

            if (isLoginMode) {
                // Save token and login
                localStorage.setItem('token', data.token);
                checkAuth();
            } else {
                // Auto-login after successful registration
                isLoginMode = true;
                authTitle.textContent = 'Login to Ticket System';
                authSubmitBtn.textContent = 'Login';
                authForm.reset();
                authError.classList.remove('hidden');
                authError.style.backgroundColor = '#d1fae5';
                authError.style.color = '#065f46';
                authError.textContent = 'Registration successful! Please login.';
            }
        } catch (err) {
            authError.classList.remove('hidden');
            authError.style.backgroundColor = '#fee2e2';
            authError.style.color = '#b91c1c';
            authError.textContent = err.message;
        }
    });

    // Logout
    logoutBtn.addEventListener('click', () => {
        localStorage.removeItem('token');
        checkAuth();
    });

    // --- Ticket Flow ---

    // API Helper with automatic auth header
    async function apiFetch(endpoint, options = {}) {
        const token = localStorage.getItem('token');
        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...(options.headers || {})
        };
        
        const res = await fetch(endpoint, { ...options, headers });
        if (res.status === 401) {
            localStorage.removeItem('token');
            checkAuth();
            throw new Error('Session expired. Please login again.');
        }
        
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
            throw new Error(data.error || 'API Request failed');
        }
        return data;
    }

    // Create Ticket
    createTicketForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const title = document.getElementById('title').value;
        const description = document.getElementById('description').value;

        try {
            ticketError.classList.add('hidden');
            await apiFetch('/tickets', {
                method: 'POST',
                body: JSON.stringify({ title, description })
            });
            createTicketForm.reset();
            fetchTickets();
        } catch (err) {
            ticketError.textContent = err.message;
            ticketError.classList.remove('hidden');
        }
    });

    // Fetch and render tickets
    async function fetchTickets() {
        try {
            const tickets = await apiFetch('/tickets');
            renderTickets(tickets || []);
        } catch (err) {
            ticketsContainer.innerHTML = `<p class="error-msg">${err.message}</p>`;
        }
    }

    function renderTickets(tickets) {
        if (tickets.length === 0) {
            ticketsContainer.innerHTML = '<div class="empty-state">No tickets found. Create one above!</div>';
            return;
        }

        // Sort tickets by creation date (newest first)
        tickets.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));

        ticketsContainer.innerHTML = tickets.map(ticket => `
            <div class="ticket-card">
                <div class="ticket-header">
                    <h3 class="ticket-title">${escapeHTML(ticket.title)}</h3>
                    <span class="status-badge status-${ticket.status}">${ticket.status.replace('_', ' ')}</span>
                </div>
                <div class="ticket-id">ID: ${ticket.id}</div>
                <div class="ticket-desc">${escapeHTML(ticket.description)}</div>
                <div class="ticket-footer">
                    <div class="ticket-date">Created: ${new Date(ticket.created_at).toLocaleString()}</div>
                    <div class="status-control">
                        <label>Update Status:</label>
                        <select class="status-select" onchange="updateTicketStatus('${ticket.id}', this.value)" 
                            ${ticket.status === 'closed' ? 'disabled' : ''}>
                            <option value="open" ${ticket.status === 'open' ? 'selected' : ''} ${ticket.status !== 'open' ? 'disabled' : ''}>Open</option>
                            <option value="in_progress" ${ticket.status === 'in_progress' ? 'selected' : ''}>In Progress</option>
                            <option value="closed" ${ticket.status === 'closed' ? 'selected' : ''}>Closed</option>
                        </select>
                    </div>
                </div>
            </div>
        `).join('');
    }

    // Global function to update ticket status
    window.updateTicketStatus = async function(id, newStatus) {
        try {
            await apiFetch(`/tickets/${id}/status`, {
                method: 'PATCH',
                body: JSON.stringify({ status: newStatus })
            });
            fetchTickets(); // Refresh list
        } catch (err) {
            alert(`Failed to update status: ${err.message}`);
            fetchTickets(); // Reset dropdown to previous state
        }
    };

    // Helper to prevent XSS
    function escapeHTML(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // Initialize
    checkAuth();
});
