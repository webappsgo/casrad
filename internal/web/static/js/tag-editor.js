// CASRAD Tag Editor - Comprehensive Metadata Editor
class TagEditor {
    constructor() {
        this.currentTrack = null;
        this.modal = null;
        this.setupModal();
    }

    setupModal() {
        // Create modal HTML
        const modalHTML = `
            <div id="tag-editor-modal" class="modal" style="display: none;">
                <div class="modal-content" style="max-width: 800px; background: var(--color-current-line); padding: var(--spacing-lg); border-radius: var(--border-radius);">
                    <div class="modal-header" style="display: flex; justify-content: space-between; margin-bottom: var(--spacing-lg);">
                        <h2>Edit Track Metadata</h2>
                        <button class="btn-close" onclick="tagEditor.close()">✕</button>
                    </div>
                    <div class="modal-body">
                        <form id="metadata-form">
                            <div class="grid grid-cols-2">
                                <!-- Basic Metadata -->
                                <div class="form-group">
                                    <label class="form-label">Title</label>
                                    <input type="text" name="title" class="form-control" required>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Artist</label>
                                    <input type="text" name="artist" class="form-control" required>
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Album</label>
                                    <input type="text" name="album" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Album Artist</label>
                                    <input type="text" name="album_artist" class="form-control">
                                </div>
                                
                                <!-- Extended Metadata -->
                                <div class="form-group">
                                    <label class="form-label">Genre</label>
                                    <input type="text" name="genre" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Year</label>
                                    <input type="number" name="year" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Track Number</label>
                                    <input type="number" name="track_number" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Disc Number</label>
                                    <input type="number" name="disc_number" class="form-control">
                                </div>
                                
                                <!-- Advanced Metadata -->
                                <div class="form-group">
                                    <label class="form-label">Composer</label>
                                    <input type="text" name="composer" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Performer</label>
                                    <input type="text" name="performer" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Label</label>
                                    <input type="text" name="label" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Catalog Number</label>
                                    <input type="text" name="catalog_number" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">ISRC</label>
                                    <input type="text" name="isrc" class="form-control">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Barcode</label>
                                    <input type="text" name="barcode" class="form-control">
                                </div>
                            </div>
                            
                            <!-- Full-width fields -->
                            <div class="form-group">
                                <label class="form-label">Comment</label>
                                <textarea name="comment" class="form-control" rows="3"></textarea>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Lyrics</label>
                                <textarea name="lyrics" class="form-control" rows="10"></textarea>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Tags (comma separated)</label>
                                <input type="text" name="tags" class="form-control" placeholder="rock, favorite, 2024">
                            </div>
                            
                            <!-- MusicBrainz IDs (read-only) -->
                            <div class="grid grid-cols-3">
                                <div class="form-group">
                                    <label class="form-label">MusicBrainz ID</label>
                                    <input type="text" name="mbid" class="form-control" readonly style="background: var(--color-selection);">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Artist MBID</label>
                                    <input type="text" name="artist_mbid" class="form-control" readonly style="background: var(--color-selection);">
                                </div>
                                <div class="form-group">
                                    <label class="form-label">Album MBID</label>
                                    <input type="text" name="album_mbid" class="form-control" readonly style="background: var(--color-selection);">
                                </div>
                            </div>
                            
                            <div class="modal-footer" style="display: flex; gap: var(--spacing-md); margin-top: var(--spacing-lg);">
                                <button type="button" class="btn btn-info" onclick="tagEditor.autoTag()">Auto-Tag with MusicBrainz</button>
                                <button type="submit" class="btn btn-primary">Save Changes</button>
                                <button type="button" class="btn btn-danger" onclick="tagEditor.close()">Cancel</button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        `;

        // Add modal to body
        const modalContainer = document.createElement('div');
        modalContainer.innerHTML = modalHTML;
        document.body.appendChild(modalContainer);

        this.modal = document.getElementById('tag-editor-modal');

        // Setup form submission
        document.getElementById('metadata-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveMetadata();
        });
    }

    async open(trackId) {
        this.currentTrack = trackId;
        
        // Fetch track metadata
        try {
            const response = await fetch(`/api/v1/tracks/${trackId}/metadata`);
            const metadata = await response.json();
            
            // Populate form
            const form = document.getElementById('metadata-form');
            for (const [key, value] of Object.entries(metadata)) {
                const input = form.elements[key];
                if (input) {
                    if (key === 'tags' && Array.isArray(value)) {
                        input.value = value.join(', ');
                    } else {
                        input.value = value || '';
                    }
                }
            }
            
            // Show modal
            this.modal.style.display = 'flex';
            this.modal.style.alignItems = 'center';
            this.modal.style.justifyContent = 'center';
            this.modal.style.position = 'fixed';
            this.modal.style.top = '0';
            this.modal.style.left = '0';
            this.modal.style.width = '100%';
            this.modal.style.height = '100%';
            this.modal.style.background = 'rgba(0, 0, 0, 0.8)';
            this.modal.style.zIndex = '10000';
            
        } catch (error) {
            console.error('Failed to load metadata:', error);
            alert('Failed to load track metadata');
        }
    }

    close() {
        this.modal.style.display = 'none';
        this.currentTrack = null;
    }

    async saveMetadata() {
        const form = document.getElementById('metadata-form');
        const formData = new FormData(form);
        
        const metadata = {};
        for (const [key, value] of formData.entries()) {
            if (key === 'tags') {
                metadata[key] = value.split(',').map(t => t.trim()).filter(t => t);
            } else if (['year', 'track_number', 'disc_number'].includes(key)) {
                metadata[key] = parseInt(value) || 0;
            } else {
                metadata[key] = value;
            }
        }
        
        try {
            const response = await fetch(`/api/v1/tracks/${this.currentTrack}/metadata`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(metadata)
            });
            
            if (response.ok) {
                this.showNotification('Metadata saved successfully', 'success');
                this.close();
                // Refresh track list if exists
                if (window.refreshTrackList) {
                    window.refreshTrackList();
                }
            } else {
                this.showNotification('Failed to save metadata', 'error');
            }
        } catch (error) {
            console.error('Save failed:', error);
            this.showNotification('Failed to save metadata', 'error');
        }
    }

    async autoTag() {
        if (!confirm('Auto-tag this track using MusicBrainz? This will overwrite some fields.')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/v1/tracks/${this.currentTrack}/autotag`, {
                method: 'POST'
            });
            
            if (response.ok) {
                this.showNotification('Auto-tagging complete', 'success');
                // Reload metadata
                this.open(this.currentTrack);
            } else {
                this.showNotification('Auto-tagging failed', 'error');
            }
        } catch (error) {
            console.error('Auto-tag failed:', error);
            this.showNotification('Auto-tagging failed', 'error');
        }
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
}

// Initialize tag editor
window.tagEditor = new TagEditor();

// Add context menu for tracks
document.addEventListener('contextmenu', (e) => {
    const trackItem = e.target.closest('[data-track-id]');
    if (trackItem) {
        e.preventDefault();
        const trackId = trackItem.dataset.trackId;
        
        // Show context menu with "Edit Tags" option
        const menu = document.createElement('div');
        menu.className = 'context-menu';
        menu.innerHTML = `
            <div class="context-menu-item" onclick="tagEditor.open(${trackId})">✏️ Edit Tags</div>
            <div class="context-menu-item" onclick="addToQueue(${trackId})">➕ Add to Queue</div>
            <div class="context-menu-item" onclick="playNext(${trackId})">⏭ Play Next</div>
        `;
        menu.style.position = 'fixed';
        menu.style.left = e.clientX + 'px';
        menu.style.top = e.clientY + 'px';
        menu.style.background = 'var(--color-current-line)';
        menu.style.border = '1px solid var(--color-border)';
        menu.style.borderRadius = 'var(--border-radius)';
        menu.style.padding = 'var(--spacing-sm)';
        menu.style.zIndex = '10001';
        
        document.body.appendChild(menu);
        
        // Remove on click outside
        setTimeout(() => {
            document.addEventListener('click', () => menu.remove(), { once: true });
        }, 100);
    }
});
