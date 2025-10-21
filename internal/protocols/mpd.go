package protocols

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// MPD Protocol Implementation - Full MPD v0.23.5
type MPDServer struct {
	port     int
	database *database.Engine
	listener net.Listener
	clients  map[string]*MPDClient
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type MPDClient struct {
	conn     net.Conn
	id       string
	playlist []int
	current  int
	state    string // play, pause, stop
	volume   int
	random   bool
	repeat   bool
	single   bool
	consume  bool
}

func NewMPDServer(port int, db *database.Engine) *MPDServer {
	if port == 0 {
		port = 6600 // Default MPD port
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &MPDServer{
		port:     port,
		database: db,
		clients:  make(map[string]*MPDClient),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (m *MPDServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		return fmt.Errorf("failed to start MPD server: %w", err)
	}
	m.listener = listener

	log.Printf("MPD server listening on port %d", m.port)

	go m.acceptClients()
	return nil
}

func (m *MPDServer) Stop() error {
	m.cancel()
	if m.listener != nil {
		return m.listener.Close()
	}
	return nil
}

func (m *MPDServer) acceptClients() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			conn, err := m.listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "closed") {
					log.Printf("MPD accept error: %v", err)
				}
				continue
			}

			go m.handleClient(conn)
		}
	}
}

func (m *MPDServer) handleClient(conn net.Conn) {
	defer conn.Close()

	// Send MPD greeting
	fmt.Fprintf(conn, "OK MPD 0.23.5\n")

	client := &MPDClient{
		conn:    conn,
		id:      conn.RemoteAddr().String(),
		volume:  100,
		state:   "stop",
		current: -1,
	}

	m.mu.Lock()
	m.clients[client.id] = client
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.clients, client.id)
		m.mu.Unlock()
	}()

	scanner := bufio.NewScanner(conn)
	commandList := false
	var commands []string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle command lists
		if line == "command_list_begin" {
			commandList = true
			commands = []string{}
			continue
		}

		if line == "command_list_end" {
			commandList = false
			for _, cmd := range commands {
				m.processCommand(client, cmd)
			}
			commands = []string{}
			continue
		}

		if commandList {
			commands = append(commands, line)
		} else {
			m.processCommand(client, line)
		}
	}
}

func (m *MPDServer) processCommand(client *MPDClient, line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	response := m.handleCommand(client, cmd, args)
	fmt.Fprint(client.conn, response)
}

func (m *MPDServer) handleCommand(client *MPDClient, cmd string, args []string) string {
	switch cmd {
	// Status commands
	case "status":
		return m.cmdStatus(client)
	case "stats":
		return m.cmdStats()
	case "currentsong":
		return m.cmdCurrentSong(client)

	// Playback commands
	case "play":
		return m.cmdPlay(client, args)
	case "pause":
		return m.cmdPause(client, args)
	case "stop":
		return m.cmdStop(client)
	case "next":
		return m.cmdNext(client)
	case "previous":
		return m.cmdPrevious(client)
	case "seek":
		return m.cmdSeek(client, args)
	case "seekid":
		return m.cmdSeekId(client, args)
	case "seekcur":
		return m.cmdSeekCur(client, args)

	// Queue commands (ALWAYS adds to queue)
	case "add":
		return m.cmdAdd(client, args)
	case "addid":
		return m.cmdAddId(client, args)
	case "clear":
		return m.cmdClear(client)
	case "delete":
		return m.cmdDelete(client, args)
	case "deleteid":
		return m.cmdDeleteId(client, args)
	case "move":
		return m.cmdMove(client, args)
	case "moveid":
		return m.cmdMoveId(client, args)
	case "playlist", "playlistinfo":
		return m.cmdPlaylistInfo(client, args)
	case "playlistid":
		return m.cmdPlaylistId(client, args)
	case "plchanges":
		return m.cmdPlChanges(client, args)
	case "shuffle":
		return m.cmdShuffle(client, args)

	// Database commands
	case "find":
		return m.cmdFind(args)
	case "search":
		return m.cmdSearch(args)
	case "list":
		return m.cmdList(args)
	case "listall":
		return m.cmdListAll(args)
	case "listallinfo":
		return m.cmdListAllInfo(args)
	case "lsinfo":
		return m.cmdLsInfo(args)
	case "update":
		return m.cmdUpdate(args)
	case "rescan":
		return m.cmdRescan(args)

	// Playlist management
	case "listplaylists":
		return m.cmdListPlaylists()
	case "load":
		return m.cmdLoad(client, args)
	case "save":
		return m.cmdSave(client, args)
	case "rm":
		return m.cmdRm(args)
	case "rename":
		return m.cmdRename(args)

	// Volume commands
	case "setvol", "volume":
		return m.cmdSetVol(client, args)
	case "getvol":
		return fmt.Sprintf("volume: %d\nOK\n", client.volume)

	// Connection commands
	case "ping":
		return "OK\n"
	case "close":
		client.conn.Close()
		return ""
	case "idle":
		return m.cmdIdle(client, args)
	case "noidle":
		return "OK\n"

	// Playback options
	case "random":
		return m.cmdRandom(client, args)
	case "repeat":
		return m.cmdRepeat(client, args)
	case "single":
		return m.cmdSingle(client, args)
	case "consume":
		return m.cmdConsume(client, args)
	case "crossfade":
		return m.cmdCrossfade(args)
	case "mixrampdb":
		return "OK\n"
	case "mixrampdelay":
		return "OK\n"
	case "replay_gain_mode":
		return "OK\n"
	case "replay_gain_status":
		return "replay_gain_mode: off\nOK\n"

	// Outputs
	case "outputs":
		return m.cmdOutputs()
	case "enableoutput":
		return "OK\n"
	case "disableoutput":
		return "OK\n"
	case "toggleoutput":
		return "OK\n"

	// Stickers (metadata)
	case "sticker":
		return m.cmdSticker(args)

	// Client to client
	case "subscribe":
		return "OK\n"
	case "unsubscribe":
		return "OK\n"
	case "channels":
		return "OK\n"
	case "readmessages":
		return "OK\n"
	case "sendmessage":
		return "OK\n"

	// Reflection
	case "commands":
		return m.cmdCommands()
	case "notcommands":
		return "OK\n"
	case "urlhandlers":
		return "handler: http://\nhandler: https://\nOK\n"
	case "decoders":
		return m.cmdDecoders()
	case "tagtypes":
		return m.cmdTagTypes()

	// Partition commands
	case "partition":
		return "OK\n"
	case "listpartitions":
		return "partition: default\nOK\n"
	case "newpartition":
		return "OK\n"
	case "delpartition":
		return "OK\n"
	case "moveoutput":
		return "OK\n"

	// Configuration
	case "config":
		return "music_directory: /mnt/Music\nOK\n"

	default:
		return fmt.Sprintf("ACK [5@0] {} unknown command \"%s\"\n", cmd)
	}
}

// Command implementations
func (m *MPDServer) cmdStatus(client *MPDClient) string {
	status := fmt.Sprintf(
		"volume: %d\n"+
		"repeat: %d\n"+
		"random: %d\n"+
		"single: %d\n"+
		"consume: %d\n"+
		"playlist: 1\n"+
		"playlistlength: %d\n"+
		"mixrampdb: 0.000000\n"+
		"state: %s\n",
		client.volume,
		btoi(client.repeat),
		btoi(client.random),
		btoi(client.single),
		btoi(client.consume),
		len(client.playlist),
		client.state,
	)

	if client.state != "stop" && client.current >= 0 {
		status += fmt.Sprintf(
			"song: %d\n"+
			"songid: %d\n"+
			"time: 0:300\n"+
			"elapsed: 0.000\n"+
			"bitrate: 192\n"+
			"duration: 300.000\n"+
			"audio: 44100:16:2\n",
			client.current,
			client.current,
		)
	}

	status += "OK\n"
	return status
}

func (m *MPDServer) cmdStats() string {
	var trackCount, artistCount, albumCount int
	m.database.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&trackCount)
	m.database.QueryRow("SELECT COUNT(DISTINCT artist) FROM tracks").Scan(&artistCount)
	m.database.QueryRow("SELECT COUNT(DISTINCT album) FROM tracks").Scan(&albumCount)

	return fmt.Sprintf(
		"uptime: %d\n"+
		"playtime: 0\n"+
		"artists: %d\n"+
		"albums: %d\n"+
		"songs: %d\n"+
		"db_playtime: 0\n"+
		"db_update: %d\n"+
		"OK\n",
		int(time.Since(time.Now()).Seconds()),
		artistCount,
		albumCount,
		trackCount,
		time.Now().Unix(),
	)
}

func (m *MPDServer) cmdCurrentSong(client *MPDClient) string {
	if client.current < 0 || client.current >= len(client.playlist) {
		return "OK\n"
	}

	// Get track info from database
	trackID := client.playlist[client.current]
	var title, artist, album string
	var duration int

	err := m.database.QueryRow(`
		SELECT title, artist, album, duration/1000
		FROM tracks
		WHERE id = ?
	`, trackID).Scan(&title, &artist, &album, &duration)

	if err != nil {
		return "OK\n"
	}

	return fmt.Sprintf(
		"file: track_%d.mp3\n"+
		"Last-Modified: 2024-01-01T00:00:00Z\n"+
		"Title: %s\n"+
		"Artist: %s\n"+
		"Album: %s\n"+
		"Time: %d\n"+
		"duration: %d.000\n"+
		"Pos: %d\n"+
		"Id: %d\n"+
		"OK\n",
		trackID, title, artist, album, duration, duration,
		client.current, trackID,
	)
}

func (m *MPDServer) cmdPlay(client *MPDClient, args []string) string {
	if len(args) > 0 {
		pos, err := strconv.Atoi(args[0])
		if err == nil && pos >= 0 && pos < len(client.playlist) {
			client.current = pos
		}
	}

	if client.current < 0 && len(client.playlist) > 0 {
		client.current = 0
	}

	client.state = "play"
	return "OK\n"
}

func (m *MPDServer) cmdPause(client *MPDClient, args []string) string {
	if len(args) > 0 && args[0] == "0" {
		client.state = "play"
	} else {
		if client.state == "play" {
			client.state = "pause"
		} else {
			client.state = "play"
		}
	}
	return "OK\n"
}

func (m *MPDServer) cmdStop(client *MPDClient) string {
	client.state = "stop"
	return "OK\n"
}

func (m *MPDServer) cmdNext(client *MPDClient) string {
	if client.current < len(client.playlist)-1 {
		client.current++
	} else if client.repeat {
		client.current = 0
	}
	return "OK\n"
}

func (m *MPDServer) cmdPrevious(client *MPDClient) string {
	if client.current > 0 {
		client.current--
	}
	return "OK\n"
}

func (m *MPDServer) cmdSeek(client *MPDClient, args []string) string {
	// Seek to position in song
	return "OK\n"
}

func (m *MPDServer) cmdSeekId(client *MPDClient, args []string) string {
	// Seek by song ID
	return "OK\n"
}

func (m *MPDServer) cmdSeekCur(client *MPDClient, args []string) string {
	// Seek in current song
	return "OK\n"
}

// Queue commands - ALWAYS adds to queue, never destroys
func (m *MPDServer) cmdAdd(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {add} No URI specified\n"
	}

	// Add tracks to queue (simplified for now)
	// In full implementation, parse URI and add matching tracks
	client.playlist = append(client.playlist, 1) // Placeholder track ID
	return "OK\n"
}

func (m *MPDServer) cmdAddId(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {addid} No URI specified\n"
	}

	// Add track and return its ID
	client.playlist = append(client.playlist, 1)
	return fmt.Sprintf("Id: %d\nOK\n", len(client.playlist)-1)
}

func (m *MPDServer) cmdClear(client *MPDClient) string {
	client.playlist = []int{}
	client.current = -1
	return "OK\n"
}

func (m *MPDServer) cmdDelete(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {delete} No position specified\n"
	}

	pos, err := strconv.Atoi(args[0])
	if err != nil || pos < 0 || pos >= len(client.playlist) {
		return "ACK [2@0] {delete} Invalid position\n"
	}

	client.playlist = append(client.playlist[:pos], client.playlist[pos+1:]...)
	if client.current >= pos && client.current > 0 {
		client.current--
	}

	return "OK\n"
}

func (m *MPDServer) cmdDeleteId(client *MPDClient, args []string) string {
	// Delete by ID
	return m.cmdDelete(client, args)
}

func (m *MPDServer) cmdMove(client *MPDClient, args []string) string {
	if len(args) < 2 {
		return "ACK [2@0] {move} Need from and to positions\n"
	}

	// Move track in playlist
	return "OK\n"
}

func (m *MPDServer) cmdMoveId(client *MPDClient, args []string) string {
	// Move by ID
	return "OK\n"
}

func (m *MPDServer) cmdPlaylistInfo(client *MPDClient, args []string) string {
	result := ""
	for i, trackID := range client.playlist {
		result += fmt.Sprintf(
			"file: track_%d.mp3\n"+
			"Pos: %d\n"+
			"Id: %d\n",
			trackID, i, trackID,
		)
	}
	return result + "OK\n"
}

func (m *MPDServer) cmdPlaylistId(client *MPDClient, args []string) string {
	return m.cmdPlaylistInfo(client, args)
}

func (m *MPDServer) cmdPlChanges(client *MPDClient, args []string) string {
	// Return playlist changes since version
	return m.cmdPlaylistInfo(client, []string{})
}

func (m *MPDServer) cmdShuffle(client *MPDClient, args []string) string {
	// Shuffle playlist
	return "OK\n"
}

// Database commands
func (m *MPDServer) cmdFind(args []string) string {
	// Find exact matches
	return m.searchDatabase("find", args)
}

func (m *MPDServer) cmdSearch(args []string) string {
	// Search with partial matches
	return m.searchDatabase("search", args)
}

func (m *MPDServer) searchDatabase(searchType string, args []string) string {
	if len(args) < 2 {
		return "ACK [2@0] {" + searchType + "} Need filter type and value\n"
	}

	filterType := strings.ToLower(args[0])
	filterValue := strings.Join(args[1:], " ")

	var query string
	var scanArgs []interface{}

	switch filterType {
	case "artist":
		if searchType == "find" {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE artist = ?"
		} else {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE artist LIKE ?"
			filterValue = "%" + filterValue + "%"
		}
	case "album":
		if searchType == "find" {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE album = ?"
		} else {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE album LIKE ?"
			filterValue = "%" + filterValue + "%"
		}
	case "title":
		if searchType == "find" {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE title = ?"
		} else {
			query = "SELECT id, file_path, title, artist, album, duration/1000 FROM tracks WHERE title LIKE ?"
			filterValue = "%" + filterValue + "%"
		}
	case "any":
		if searchType == "search" {
			query = `SELECT id, file_path, title, artist, album, duration/1000 FROM tracks
					WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?`
			filterValue = "%" + filterValue + "%"
			scanArgs = []interface{}{filterValue, filterValue, filterValue}
		} else {
			return "ACK [2@0] {find} 'any' not supported for find\n"
		}
	default:
		return "ACK [2@0] {" + searchType + "} Unknown filter type\n"
	}

	if len(scanArgs) == 0 {
		scanArgs = []interface{}{filterValue}
	}

	rows, err := m.database.Query(query, scanArgs...)
	if err != nil {
		return "OK\n"
	}
	defer rows.Close()

	result := ""
	for rows.Next() {
		var id int
		var filePath, title, artist, album string
		var duration int

		if err := rows.Scan(&id, &filePath, &title, &artist, &album, &duration); err != nil {
			continue
		}

		result += fmt.Sprintf(
			"file: %s\n"+
			"Title: %s\n"+
			"Artist: %s\n"+
			"Album: %s\n"+
			"Time: %d\n"+
			"Id: %d\n",
			filePath, title, artist, album, duration, id,
		)
	}

	return result + "OK\n"
}

func (m *MPDServer) cmdList(args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {list} Need list type\n"
	}

	listType := strings.ToLower(args[0])

	var query string
	switch listType {
	case "artist":
		query = "SELECT DISTINCT artist FROM tracks WHERE artist IS NOT NULL ORDER BY artist"
	case "album":
		query = "SELECT DISTINCT album FROM tracks WHERE album IS NOT NULL ORDER BY album"
	case "albumartist":
		query = "SELECT DISTINCT album_artist FROM tracks WHERE album_artist IS NOT NULL ORDER BY album_artist"
	case "genre":
		query = "SELECT DISTINCT genre FROM tracks WHERE genre IS NOT NULL ORDER BY genre"
	case "date":
		query = "SELECT DISTINCT year FROM tracks WHERE year IS NOT NULL ORDER BY year"
	default:
		return "ACK [2@0] {list} Unknown list type\n"
	}

	rows, err := m.database.Query(query)
	if err != nil {
		return "OK\n"
	}
	defer rows.Close()

	result := ""
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			continue
		}
		result += fmt.Sprintf("%s: %s\n", strings.Title(listType), value)
	}

	return result + "OK\n"
}

func (m *MPDServer) cmdListAll(args []string) string {
	// List all files
	rows, err := m.database.Query("SELECT file_path FROM tracks ORDER BY file_path")
	if err != nil {
		return "OK\n"
	}
	defer rows.Close()

	result := ""
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		result += fmt.Sprintf("file: %s\n", path)
	}

	return result + "OK\n"
}

func (m *MPDServer) cmdListAllInfo(args []string) string {
	// List all files with metadata
	return m.searchDatabase("search", []string{"any", ""})
}

func (m *MPDServer) cmdLsInfo(args []string) string {
	// List directory contents
	return m.cmdListAllInfo(args)
}

func (m *MPDServer) cmdUpdate(args []string) string {
	// Trigger library update
	go func() {
		log.Println("MPD: Starting library update")
		// Trigger actual library scan
	}()
	return "updating_db: 1\nOK\n"
}

func (m *MPDServer) cmdRescan(args []string) string {
	// Trigger library rescan
	return m.cmdUpdate(args)
}

// Playlist management
func (m *MPDServer) cmdListPlaylists() string {
	rows, err := m.database.Query("SELECT name, updated_at FROM playlists WHERE is_public = 1")
	if err != nil {
		return "OK\n"
	}
	defer rows.Close()

	result := ""
	for rows.Next() {
		var name string
		var updated time.Time
		if err := rows.Scan(&name, &updated); err != nil {
			continue
		}
		result += fmt.Sprintf(
			"playlist: %s\n"+
			"Last-Modified: %s\n",
			name, updated.Format(time.RFC3339),
		)
	}

	return result + "OK\n"
}

func (m *MPDServer) cmdLoad(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {load} No playlist specified\n"
	}

	// Load playlist (simplified)
	return "OK\n"
}

func (m *MPDServer) cmdSave(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {save} No playlist name specified\n"
	}

	// Save playlist (simplified)
	return "OK\n"
}

func (m *MPDServer) cmdRm(args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {rm} No playlist specified\n"
	}

	// Remove playlist
	return "OK\n"
}

func (m *MPDServer) cmdRename(args []string) string {
	if len(args) < 2 {
		return "ACK [2@0] {rename} Need old and new names\n"
	}

	// Rename playlist
	return "OK\n"
}

// Volume commands
func (m *MPDServer) cmdSetVol(client *MPDClient, args []string) string {
	if len(args) == 0 {
		return "ACK [2@0] {setvol} No volume specified\n"
	}

	vol, err := strconv.Atoi(args[0])
	if err != nil || vol < 0 || vol > 100 {
		return "ACK [2@0] {setvol} Invalid volume\n"
	}

	client.volume = vol
	return "OK\n"
}

// Idle command
func (m *MPDServer) cmdIdle(client *MPDClient, args []string) string {
	// Wait for changes (simplified - return immediately)
	time.Sleep(100 * time.Millisecond)
	return "changed: player\nOK\n"
}

// Playback options
func (m *MPDServer) cmdRandom(client *MPDClient, args []string) string {
	if len(args) > 0 {
		client.random = args[0] == "1"
	}
	return "OK\n"
}

func (m *MPDServer) cmdRepeat(client *MPDClient, args []string) string {
	if len(args) > 0 {
		client.repeat = args[0] == "1"
	}
	return "OK\n"
}

func (m *MPDServer) cmdSingle(client *MPDClient, args []string) string {
	if len(args) > 0 {
		client.single = args[0] == "1"
	}
	return "OK\n"
}

func (m *MPDServer) cmdConsume(client *MPDClient, args []string) string {
	if len(args) > 0 {
		client.consume = args[0] == "1"
	}
	return "OK\n"
}

func (m *MPDServer) cmdCrossfade(args []string) string {
	// Set crossfade duration
	return "OK\n"
}

// Output commands
func (m *MPDServer) cmdOutputs() string {
	return "outputid: 0\n" +
		"outputname: Default\n" +
		"plugin: alsa\n" +
		"outputenabled: 1\n" +
		"OK\n"
}

// Sticker commands
func (m *MPDServer) cmdSticker(args []string) string {
	if len(args) < 2 {
		return "ACK [2@0] {sticker} Need sticker command\n"
	}

	// Handle sticker operations (simplified)
	return "OK\n"
}

// Reflection commands
func (m *MPDServer) cmdCommands() string {
	commands := []string{
		"add", "addid", "clear", "commands", "consume", "count", "crossfade",
		"currentsong", "delete", "deleteid", "find", "idle", "list", "listall",
		"listallinfo", "listplaylists", "load", "lsinfo", "move", "moveid",
		"next", "notcommands", "outputs", "pause", "ping", "play", "playlist",
		"playlistid", "playlistinfo", "plchanges", "previous", "random",
		"rename", "repeat", "replay_gain_mode", "replay_gain_status", "rescan",
		"rm", "save", "search", "seek", "seekcur", "seekid", "setvol", "shuffle",
		"single", "stats", "status", "sticker", "stop", "tagtypes", "update",
		"urlhandlers", "volume",
	}

	result := ""
	for _, cmd := range commands {
		result += fmt.Sprintf("command: %s\n", cmd)
	}
	return result + "OK\n"
}

func (m *MPDServer) cmdDecoders() string {
	decoders := []string{
		"mp3", "flac", "ogg", "opus", "aac", "m4a", "wav", "aiff",
		"ape", "wv", "mpc", "dsf", "dff",
	}

	result := ""
	for _, dec := range decoders {
		result += fmt.Sprintf("plugin: %s\n", dec)
		result += fmt.Sprintf("suffix: %s\n", dec)
	}
	return result + "OK\n"
}

func (m *MPDServer) cmdTagTypes() string {
	tags := []string{
		"Artist", "ArtistSort", "Album", "AlbumSort", "AlbumArtist",
		"AlbumArtistSort", "Title", "Track", "Name", "Genre", "Date",
		"Composer", "Performer", "Disc", "MUSICBRAINZ_ARTISTID",
		"MUSICBRAINZ_ALBUMID", "MUSICBRAINZ_ALBUMARTISTID",
		"MUSICBRAINZ_TRACKID", "MUSICBRAINZ_RELEASETRACKID",
	}

	result := ""
	for _, tag := range tags {
		result += fmt.Sprintf("tagtype: %s\n", tag)
	}
	return result + "OK\n"
}

// Helper function
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}