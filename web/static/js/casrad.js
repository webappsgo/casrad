// CASRAD Web Client
class CASRADPlayer {
    constructor() {
        this.audio = document.getElementById('audio-player');
        this.currentTrack = null;
        this.queue = [];
        this.queueIndex = 0;
        this.isPlaying = false;

        this.initializeEventListeners();
        this.loadLibrary();
        this.startMetricsUpdates();
    }

    initializeEventListeners() {
        // Player controls
        document.getElementById('play-btn').addEventListener('click', () => this.togglePlayPause());
        document.getElementById('prev-btn').addEventListener('click', () => this.playPrevious());
        document.getElementById('next-btn').addEventListener('click', () => this.playNext());

        // Audio events
        this.audio.addEventListener('timeupdate', () => this.updateProgress());
        this.audio.addEventListener('ended', () => this.playNext());
        this.audio.addEventListener('loadedmetadata', () => this.updateDuration());

        // Progress bar click
        document.querySelector('.player .progress').addEventListener('click', (e) => {
            const rect = e.currentTarget.getBoundingClientRect();
            const percent = (e.clientX - rect.left) / rect.width;
            this.audio.currentTime = this.audio.duration * percent;
        });
    }

    async scanLibrary() {
        try {
            const response = await fetch('/api/v1/library/scan', { method: 'POST' });
            const result = await response.json();
            console.log('Library scan started:', result);
            this.showToast('Library scan started', 'info');
        } catch (error) {
            console.error('Failed to start library scan:', error);
        }
    }

    async loadLibrary() {
        try {
            // Load tracks
            const tracksResponse = await fetch('/api/v1/tracks');
            const tracks = await tracksResponse.json();
            this.displayTracks(tracks);
            document.getElementById('total-tracks').textContent = tracks.length;

            // Load albums
            const albumsResponse = await fetch('/api/v1/albums');
            const albums = await albumsResponse.json();
            this.displayAlbums(albums);
            document.getElementById('total-albums').textContent = albums.length;

            // Load artists
            const artistsResponse = await fetch('/api/v1/artists');
            const artists = await artistsResponse.json();
            document.getElementById('total-artists').textContent = artists.length;

            // Load playlists
            const playlistsResponse = await fetch('/api/v1/playlists');
            const playlists = await playlistsResponse.json();
            document.getElementById('total-playlists').textContent = playlists.length;

        } catch (error) {
            console.error('Failed to load library:', error);
            this.showToast('Failed to load library', 'error');
        }
    }

    displayTracks(tracks) {
        const trackList = document.getElementById('track-list');
        trackList.innerHTML = '';

        tracks.forEach((track, index) => {
            const li = document.createElement('li');
            li.className = 'track-item';
            li.innerHTML = `
                <span class="track-number">${index + 1}</span>
                <div class="track-info">
                    <div class="track-title">${track.title || 'Unknown Title'}</div>
                    <div class="track-artist">${track.artist || 'Unknown Artist'}</div>
                </div>
                <span class="track-duration">${this.formatDuration(track.duration)}</span>
            `;
            li.addEventListener('click', () => this.playTrack(track));
            trackList.appendChild(li);
        });

        // Add tracks to queue
        this.queue = tracks;
    }

    displayAlbums(albums) {
        const albumGrid = document.getElementById('album-grid');
        albumGrid.innerHTML = '';

        albums.forEach(album => {
            const div = document.createElement('div');
            div.className = 'album-card';
            div.innerHTML = `
                <div class="album-cover" style="background: linear-gradient(135deg,
                    hsl(${Math.random() * 360}, 70%, 50%),
                    hsl(${Math.random() * 360}, 70%, 60%));">
                </div>
                <div class="album-info">
                    <div class="album-title">${album.title || 'Unknown Album'}</div>
                    <div class="album-artist">${album.artist || 'Various Artists'}</div>
                </div>
            `;
            div.addEventListener('click', () => this.loadAlbum(album.id));
            albumGrid.appendChild(div);
        });
    }

    async playTrack(track) {
        this.currentTrack = track;

        // Update UI
        document.getElementById('current-title').textContent = track.title || 'Unknown Title';
        document.getElementById('current-artist').textContent = track.artist || 'Unknown Artist';
        document.getElementById('current-album').textContent = track.album || 'Unknown Album';

        // Update active track in list
        document.querySelectorAll('.track-item').forEach(item => {
            item.classList.remove('active');
        });
        event.currentTarget.classList.add('active');

        // Load and play audio
        try {
            const streamUrl = `/api/v1/stream/${track.id}`;
            this.audio.src = streamUrl;
            await this.audio.play();
            this.isPlaying = true;
            this.updatePlayButton();

            // Record play
            fetch(`/api/v1/tracks/${track.id}/play`, { method: 'POST' });
        } catch (error) {
            console.error('Playback failed:', error);
            this.showToast('Playback failed', 'error');
        }
    }

    togglePlayPause() {
        if (this.isPlaying) {
            this.audio.pause();
            this.isPlaying = false;
        } else {
            if (this.audio.src) {
                this.audio.play();
                this.isPlaying = true;
            } else if (this.queue.length > 0) {
                this.playTrack(this.queue[0]);
            }
        }
        this.updatePlayButton();
    }

    playNext() {
        if (this.queue.length > 0) {
            this.queueIndex = (this.queueIndex + 1) % this.queue.length;
            this.playTrack(this.queue[this.queueIndex]);
        }
    }

    playPrevious() {
        if (this.queue.length > 0) {
            this.queueIndex = (this.queueIndex - 1 + this.queue.length) % this.queue.length;
            this.playTrack(this.queue[this.queueIndex]);
        }
    }

    updatePlayButton() {
        const playBtn = document.getElementById('play-btn');
        playBtn.innerHTML = this.isPlaying ? '⏸' : '▶';
    }

    updateProgress() {
        if (this.audio.duration) {
            const percent = (this.audio.currentTime / this.audio.duration) * 100;
            document.getElementById('player-progress').style.width = `${percent}%`;
            document.getElementById('current-time').textContent = this.formatTime(this.audio.currentTime);
        }
    }

    updateDuration() {
        document.getElementById('total-time').textContent = this.formatTime(this.audio.duration);
    }

    formatTime(seconds) {
        if (!seconds || isNaN(seconds)) return '0:00';
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }

    formatDuration(milliseconds) {
        if (!milliseconds) return '0:00';
        return this.formatTime(milliseconds / 1000);
    }

    async loadAlbum(albumId) {
        try {
            const response = await fetch(`/api/v1/albums/${albumId}/tracks`);
            const tracks = await response.json();
            this.queue = tracks;
            this.queueIndex = 0;
            if (tracks.length > 0) {
                this.playTrack(tracks[0]);
            }
        } catch (error) {
            console.error('Failed to load album:', error);
        }
    }

    async startMetricsUpdates() {
        // Update metrics every 5 seconds
        setInterval(async () => {
            try {
                const response = await fetch('/api/v1/metrics');
                const metrics = await response.json();

                // Update UI with metrics
                if (metrics['system.cpu.usage']) {
                    document.getElementById('cpu-usage').textContent =
                        Math.round(metrics['system.cpu.usage'].value) + '%';
                }
                if (metrics['system.memory.alloc']) {
                    const mb = Math.round(metrics['system.memory.alloc'].value / 1024 / 1024);
                    document.getElementById('memory-usage').textContent = mb + 'MB';
                }
                if (metrics['streaming.streams.active']) {
                    document.getElementById('active-streams').textContent =
                        metrics['streaming.streams.active'].value;
                }
                if (metrics['application.sessions.active']) {
                    document.getElementById('online-users').textContent =
                        metrics['application.sessions.active'].value;
                }
            } catch (error) {
                console.error('Failed to fetch metrics:', error);
            }
        }, 5000);
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type} fade-in`;
        toast.textContent = message;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 3000);
    }
}

// Initialize player when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.casradPlayer = new CASRADPlayer();

    // Make CASRAD globally accessible for HTML buttons
    window.CASRAD = {
        scanLibrary: () => window.casradPlayer.scanLibrary()
    };

    // Add keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if (e.target.tagName === 'INPUT') return;

        switch(e.key) {
            case ' ':
                e.preventDefault();
                window.casradPlayer.togglePlayPause();
                break;
            case 'ArrowRight':
                window.casradPlayer.playNext();
                break;
            case 'ArrowLeft':
                window.casradPlayer.playPrevious();
                break;
        }
    });

    // Trigger initial scan on load
    window.casradPlayer.scanLibrary();
});

// WebSocket for real-time updates (optional)
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);

        switch(data.type) {
            case 'now_playing':
                // Update now playing display for other users
                break;
            case 'library_update':
                // Reload library if changes detected
                window.casradPlayer.loadLibrary();
                break;
            case 'metrics':
                // Real-time metrics update
                break;
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        // Reconnect after 5 seconds
        setTimeout(connectWebSocket, 5000);
    };
}

// Uncomment to enable WebSocket
// connectWebSocket();