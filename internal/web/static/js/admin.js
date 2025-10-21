// CASRAD Admin Interface - Comprehensive Implementation
class CASRADAdmin {
    constructor() {
        this.apiBase = '/api/v1/admin';
        this.init();
    }

    async init() {
        await this.loadDashboard();
        this.setupEventListeners();
        this.startMetricsPolling();
    }

    async loadDashboard() {
        try {
            const stats = await this.fetchStats();
            this.updateDashboard(stats);
        } catch (e) {
            console.error('Failed to load dashboard:', e);
        }
    }

    async fetchStats() {
        const response = await fetch(`${this.apiBase}/stats`);
        return await response.json();
    }

    updateDashboard(stats) {
        document.getElementById('stat-users').textContent = stats.users || 0;
        document.getElementById('stat-tracks').textContent = stats.tracks || 0;
        document.getElementById('stat-streams').textContent = stats.activeStreams || 0;
        document.getElementById('stat-storage').textContent = this.formatBytes(stats.storageUsed);
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    setupEventListeners() {
        // User management
        document.addEventListener('click', (e) => {
            if (e.target.matches('.btn-create-user')) {
                this.showCreateUserModal();
            } else if (e.target.matches('.btn-edit-user')) {
                this.showEditUserModal(e.target.dataset.userId);
            } else if (e.target.matches('.btn-delete-user')) {
                this.confirmDeleteUser(e.target.dataset.userId);
            }
        });

        // Library management
        document.addEventListener('click', (e) => {
            if (e.target.matches('.btn-scan-library')) {
                this.scanLibrary();
            } else if (e.target.matches('.btn-add-directory')) {
                this.showAddDirectoryModal();
            }
        });

        // Backup/restore
        document.addEventListener('click', (e) => {
            if (e.target.matches('.btn-create-backup')) {
                this.createBackup();
            } else if (e.target.matches('.btn-restore-backup')) {
                this.showRestoreModal();
            }
        });

        // Migration
        document.addEventListener('click', (e) => {
            if (e.target.matches('.btn-start-migration')) {
                this.startMigration();
            }
        });
    }

    async scanLibrary() {
        try {
            const response = await fetch(`${this.apiBase}/library/scan`, {
                method: 'POST'
            });
            if (response.ok) {
                this.showNotification('Library scan started', 'success');
            }
        } catch (e) {
            this.showNotification('Failed to start scan', 'error');
        }
    }

    async createBackup() {
        try {
            const response = await fetch(`${this.apiBase}/backup`, {
                method: 'POST'
            });
            if (response.ok) {
                const result = await response.json();
                this.showNotification(`Backup created: ${result.filename}`, 'success');
            }
        } catch (e) {
            this.showNotification('Backup failed', 'error');
        }
    }

    async startMigration() {
        const form = document.getElementById('migration-form');
        const formData = new FormData(form);
        
        try {
            const response = await fetch(`${this.apiBase}/migrate`, {
                method: 'POST',
                body: formData
            });
            if (response.ok) {
                this.showNotification('Migration started', 'success');
                this.pollMigrationStatus();
            }
        } catch (e) {
            this.showNotification('Migration failed to start', 'error');
        }
    }

    async pollMigrationStatus() {
        const interval = setInterval(async () => {
            const response = await fetch(`${this.apiBase}/migrate/status`);
            const status = await response.json();
            
            this.updateMigrationProgress(status);
            
            if (status.status === 'completed' || status.status === 'failed') {
                clearInterval(interval);
            }
        }, 1000);
    }

    updateMigrationProgress(status) {
        const progressBar = document.getElementById('migration-progress');
        const progressText = document.getElementById('migration-text');
        
        if (progressBar && progressText) {
            const percent = (status.itemsMigrated / status.itemsTotal) * 100;
            progressBar.style.width = percent + '%';
            progressText.textContent = `${status.itemsMigrated} / ${status.itemsTotal} items`;
        }
    }

    startMetricsPolling() {
        setInterval(() => {
            this.loadDashboard();
        }, 5000);
    }

    showNotification(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 3000);
    }

    showCreateUserModal() {
        // Implementation for user creation modal
        console.log('Show create user modal');
    }

    showEditUserModal(userId) {
        // Implementation for user edit modal
        console.log('Show edit user modal:', userId);
    }

    confirmDeleteUser(userId) {
        if (confirm('Are you sure you want to delete this user?')) {
            this.deleteUser(userId);
        }
    }

    async deleteUser(userId) {
        try {
            const response = await fetch(`${this.apiBase}/users/${userId}`, {
                method: 'DELETE'
            });
            if (response.ok) {
                this.showNotification('User deleted', 'success');
                this.loadDashboard();
            }
        } catch (e) {
            this.showNotification('Failed to delete user', 'error');
        }
    }

    showAddDirectoryModal() {
        // Implementation for add directory modal
        console.log('Show add directory modal');
    }

    showRestoreModal() {
        // Implementation for restore modal
        console.log('Show restore modal');
    }
}

// Initialize admin interface if on admin page
if (window.location.pathname.startsWith('/admin')) {
    window.addEventListener('DOMContentLoaded', () => {
        window.admin = new CASRADAdmin();
    });
}
