// CASRAD Audio Player - Comprehensive Implementation
class CASRADPlayer {
    constructor() {
        this.audio = new Audio();
        this.queue = [];
        this.currentIndex = -1;
        this.shuffle = false;
        this.repeat = 'off'; // off, one, all
        this.volume = 0.7;
        this.crossfadeDuration = 5; // seconds
        this.setupEventListeners();
        this.loadState();
    }

    setupEventListeners() {
        this.audio.addEventListener('timeupdate', () => this.onTimeUpdate());
        this.audio.addEventListener('ended', () => this.onTrackEnded());
        this.audio.addEventListener('error', (e) => this.onError(e));
        this.audio.addEventListener('loadedmetadata', () => this.onMetadataLoaded());
        this.audio.addEventListener('play', () => this.onPlay());
        this.audio.addEventListener('pause', () => this.onPause());
        this.audio.addEventListener('volumechange', () => this.onVolumeChange());
        
        // Handle crossfade for smooth transitions
        this.nextAudio = new Audio();
        this.setupCrossfade();
    }

    setupCrossfade() {
        this.audio.addEventListener('timeupdate', () => {
            const remaining = this.audio.duration - this.audio.currentTime;
            if (remaining <= this.crossfadeDuration && remaining > 0) {
                if (!this.nextAudio.src && this.hasNext()) {
                    const nextTrack = this.getNextTrack();
                    this.nextAudio.src = nextTrack.url;
                    this.nextAudio.volume = 0;
                    this.nextAudio.load();
                }
                
                // Crossfade volume
                const fadeProgress = (this.crossfadeDuration - remaining) / this.crossfadeDuration;
                this.audio.volume = this.volume * (1 - fadeProgress);
                this.nextAudio.volume = this.volume * fadeProgress;
                
                if (!this.nextAudio.paused && this.nextAudio.currentTime === 0) {
                    this.nextAudio.play().catch(e => console.error('Crossfade play error:', e));
                }
            }
        });
    }

    // Queue management
    addToQueue(track) {
        this.queue.push(track);
        this.saveState();
        this.emit('queueUpdated', this.queue);
    }

    addNextInQueue(track) {
        this.queue.splice(this.currentIndex + 1, 0, track);
        this.saveState();
        this.emit('queueUpdated', this.queue);
    }

    removeFromQueue(index) {
        this.queue.splice(index, 1);
        if (index < this.currentIndex) {
            this.currentIndex--;
        }
        this.saveState();
        this.emit('queueUpdated', this.queue);
    }

    clearQueue() {
        this.queue = [];
        this.currentIndex = -1;
        this.stop();
        this.saveState();
        this.emit('queueUpdated', this.queue);
    }

    moveInQueue(from, to) {
        const track = this.queue.splice(from, 1)[0];
        this.queue.splice(to, 0, track);
        if (from === this.currentIndex) {
            this.currentIndex = to;
        } else if (from < this.currentIndex && to >= this.currentIndex) {
            this.currentIndex--;
        } else if (from > this.currentIndex && to <= this.currentIndex) {
            this.currentIndex++;
        }
        this.saveState();
        this.emit('queueUpdated', this.queue);
    }

    // Playback control
    play(index) {
        if (index !== undefined) {
            this.currentIndex = index;
        }
        if (this.currentIndex < 0 || this.currentIndex >= this.queue.length) {
            this.currentIndex = 0;
        }
        const track = this.queue[this.currentIndex];
        if (!track) return;

        this.audio.src = track.url;
        this.audio.play().catch(e => {
            console.error('Play error:', e);
            this.emit('error', { message: 'Failed to play track', error: e });
        });

        this.updateNowPlaying(track);
        this.scrobble(track);
    }

    pause() {
        this.audio.pause();
    }

    stop() {
        this.audio.pause();
        this.audio.currentTime = 0;
    }

    next() {
        if (!this.hasNext()) {
            if (this.repeat === 'all') {
                this.currentIndex = 0;
                this.play();
            }
            return;
        }

        if (this.shuffle) {
            this.currentIndex = this.getRandomIndex();
        } else {
            this.currentIndex++;
        }
        this.play();
    }

    previous() {
        if (this.audio.currentTime > 3) {
            this.audio.currentTime = 0;
            return;
        }

        if (this.currentIndex > 0) {
            this.currentIndex--;
            this.play();
        }
    }

    seek(time) {
        this.audio.currentTime = time;
    }

    seekPercent(percent) {
        this.audio.currentTime = (percent / 100) * this.audio.duration;
    }

    setVolume(volume) {
        this.volume = Math.max(0, Math.min(1, volume));
        this.audio.volume = this.volume;
        this.saveState();
    }

    toggleShuffle() {
        this.shuffle = !this.shuffle;
        this.saveState();
        this.emit('shuffleChanged', this.shuffle);
    }

    cycleRepeat() {
        const modes = ['off', 'one', 'all'];
        const currentIndex = modes.indexOf(this.repeat);
        this.repeat = modes[(currentIndex + 1) % modes.length];
        this.saveState();
        this.emit('repeatChanged', this.repeat);
    }

    // State management
    saveState() {
        const state = {
            queue: this.queue.map(t => ({ id: t.id, url: t.url })),
            currentIndex: this.currentIndex,
            shuffle: this.shuffle,
            repeat: this.repeat,
            volume: this.volume,
            position: this.audio.currentTime
        };
        fetch('/api/v1/player/state', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(state)
        }).catch(e => console.error('Failed to save state:', e));
    }

    async loadState() {
        try {
            const response = await fetch('/api/v1/player/state');
            if (response.ok) {
                const state = await response.json();
                this.queue = state.queue || [];
                this.currentIndex = state.currentIndex || -1;
                this.shuffle = state.shuffle || false;
                this.repeat = state.repeat || 'off';
                this.volume = state.volume || 0.7;
                if (state.position) {
                    this.audio.currentTime = state.position;
                }
                this.emit('stateLoaded', state);
            }
        } catch (e) {
            console.error('Failed to load state:', e);
        }
    }

    // Helper methods
    hasNext() {
        return this.currentIndex < this.queue.length - 1;
    }

    getNextTrack() {
        if (this.shuffle) {
            const index = this.getRandomIndex();
            return this.queue[index];
        }
        return this.queue[this.currentIndex + 1];
    }

    getRandomIndex() {
        let index;
        do {
            index = Math.floor(Math.random() * this.queue.length);
        } while (index === this.currentIndex && this.queue.length > 1);
        return index;
    }

    updateNowPlaying(track) {
        if ('mediaSession' in navigator) {
            navigator.mediaSession.metadata = new MediaMetadata({
                title: track.title,
                artist: track.artist,
                album: track.album,
                artwork: track.coverArt ? [
                    { src: track.coverArt, sizes: '512x512', type: 'image/jpeg' }
                ] : []
            });

            navigator.mediaSession.setActionHandler('play', () => this.audio.play());
            navigator.mediaSession.setActionHandler('pause', () => this.pause());
            navigator.mediaSession.setActionHandler('previoustrack', () => this.previous());
            navigator.mediaSession.setActionHandler('nexttrack', () => this.next());
            navigator.mediaSession.setActionHandler('seekto', (details) => this.seek(details.seekTime));
        }

        document.title = `${track.title} - ${track.artist} | CASRAD`;
    }

    async scrobble(track) {
        // Scrobble after 50% playback or 4 minutes
        const scrobbleTime = Math.min(track.duration * 0.5, 240);
        setTimeout(() => {
            fetch('/api/v1/scrobble', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ track_id: track.id, timestamp: Date.now() })
            }).catch(e => console.error('Scrobble failed:', e));
        }, scrobbleTime * 1000);
    }

    // Event handlers
    onTimeUpdate() {
        const percent = (this.audio.currentTime / this.audio.duration) * 100;
        this.emit('timeUpdate', {
            current: this.audio.currentTime,
            duration: this.audio.duration,
            percent: percent
        });
    }

    onTrackEnded() {
        if (this.repeat === 'one') {
            this.play(this.currentIndex);
        } else {
            this.next();
        }
    }

    onError(e) {
        console.error('Audio error:', e);
        this.emit('error', { error: e, track: this.queue[this.currentIndex] });
        // Try next track after error
        setTimeout(() => this.next(), 1000);
    }

    onMetadataLoaded() {
        this.emit('metadataLoaded', {
            duration: this.audio.duration,
            track: this.queue[this.currentIndex]
        });
    }

    onPlay() {
        this.emit('play', this.queue[this.currentIndex]);
    }

    onPause() {
        this.emit('pause', this.queue[this.currentIndex]);
    }

    onVolumeChange() {
        this.emit('volumeChange', this.audio.volume);
    }

    // Event emitter
    emit(event, data) {
        const customEvent = new CustomEvent(`player:${event}`, { detail: data });
        document.dispatchEvent(customEvent);
    }

    on(event, handler) {
        document.addEventListener(`player:${event}`, (e) => handler(e.detail));
    }
}

// Export for use
window.CASRADPlayer = CASRADPlayer;
