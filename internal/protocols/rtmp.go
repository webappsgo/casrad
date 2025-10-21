package protocols

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/casapps/casrad/internal/media"
)

// RTMPServer implements an RTMP streaming server
type RTMPServer struct {
	db         *database.Engine
	port       int
	listener   net.Listener
	ffmpeg     *media.FFMPEGManager
	transcoder *media.Transcoder
	streams    map[string]*RTMPStream
	mu         sync.RWMutex
	enabled    bool
}

// RTMPStream represents an active RTMP stream
type RTMPStream struct {
	streamKey  string
	mountPoint string
	conn       net.Conn
	userID     int

	// Stream info
	videoCodec   string
	audioCodec   string
	videoBitrate int
	audioBitrate int
	width        int
	height       int
	fps          int

	// Statistics
	bytesReceived int64
	startTime     time.Time
	viewers       int

	// State
	isActive bool
	mu       sync.RWMutex
}

// RTMP message types
const (
	RTMP_MSG_ChunkSize         = 1
	RTMP_MSG_Abort             = 2
	RTMP_MSG_Acknowledgement   = 3
	RTMP_MSG_UserControl       = 4
	RTMP_MSG_WindowAckSize     = 5
	RTMP_MSG_SetPeerBandwidth  = 6
	RTMP_MSG_Audio             = 8
	RTMP_MSG_Video             = 9
	RTMP_MSG_DataAMF3          = 15
	RTMP_MSG_SharedObjectAMF3  = 16
	RTMP_MSG_CommandAMF3       = 17
	RTMP_MSG_DataAMF0          = 18
	RTMP_MSG_SharedObjectAMF0  = 19
	RTMP_MSG_CommandAMF0       = 20
	RTMP_MSG_Aggregate         = 22
)

// NewRTMPServer creates a new RTMP server
func NewRTMPServer(port int, db *database.Engine, ffmpeg *media.FFMPEGManager, transcoder *media.Transcoder) *RTMPServer {
	return &RTMPServer{
		db:         db,
		port:       port,
		ffmpeg:     ffmpeg,
		transcoder: transcoder,
		streams:    make(map[string]*RTMPStream),
		enabled:    true,
	}
}

// Start starts the RTMP server
func (r *RTMPServer) Start() error {
	if !r.enabled {
		return fmt.Errorf("RTMP server is disabled")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		return fmt.Errorf("failed to start RTMP server: %w", err)
	}

	r.listener = listener
	log.Printf("RTMP server listening on port %d", r.port)

	go r.acceptConnections()

	return nil
}

// Stop stops the RTMP server
func (r *RTMPServer) Stop() error {
	r.enabled = false

	if r.listener != nil {
		return r.listener.Close()
	}

	// Close all active streams
	r.mu.Lock()
	for _, stream := range r.streams {
		stream.Close()
	}
	r.streams = make(map[string]*RTMPStream)
	r.mu.Unlock()

	return nil
}

// acceptConnections accepts incoming RTMP connections
func (r *RTMPServer) acceptConnections() {
	for r.enabled {
		conn, err := r.listener.Accept()
		if err != nil {
			if r.enabled {
				log.Printf("RTMP accept error: %v", err)
			}
			continue
		}

		go r.handleConnection(conn)
	}
}

// handleConnection handles an RTMP connection
func (r *RTMPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Perform RTMP handshake
	if err := r.performHandshake(conn); err != nil {
		log.Printf("RTMP handshake failed: %v", err)
		return
	}

	// Create stream
	stream := &RTMPStream{
		conn:      conn,
		startTime: time.Now(),
		isActive:  true,
	}

	// Handle RTMP messages
	for stream.isActive {
		if err := r.handleMessage(stream); err != nil {
			if err != io.EOF {
				log.Printf("RTMP message error: %v", err)
			}
			break
		}
	}

	// Clean up stream
	r.removeStream(stream)
}

// performHandshake performs RTMP handshake
func (r *RTMPServer) performHandshake(conn net.Conn) error {
	// C0 and C1
	c0c1 := make([]byte, 1537)
	if _, err := io.ReadFull(conn, c0c1); err != nil {
		return err
	}

	// Check version
	if c0c1[0] != 3 {
		return fmt.Errorf("unsupported RTMP version: %d", c0c1[0])
	}

	// S0 and S1
	s0s1 := make([]byte, 1537)
	s0s1[0] = 3 // Version
	// Set timestamp
	binary.BigEndian.PutUint32(s0s1[1:5], uint32(time.Now().Unix()))
	// Echo back C1
	copy(s0s1[9:], c0c1[1:])

	if _, err := conn.Write(s0s1); err != nil {
		return err
	}

	// S2 - echo C1
	if _, err := conn.Write(c0c1[1:]); err != nil {
		return err
	}

	// C2
	c2 := make([]byte, 1536)
	if _, err := io.ReadFull(conn, c2); err != nil {
		return err
	}

	return nil
}

// handleMessage handles an RTMP message
func (r *RTMPServer) handleMessage(stream *RTMPStream) error {
	// Read basic header
	basicHeader := make([]byte, 1)
	if _, err := io.ReadFull(stream.conn, basicHeader); err != nil {
		return err
	}

	fmt := (basicHeader[0] >> 6) & 0x03
	csid := basicHeader[0] & 0x3f

	// Read chunk stream ID if needed
	if csid == 0 {
		// 2 byte form
		b := make([]byte, 1)
		if _, err := io.ReadFull(stream.conn, b); err != nil {
			return err
		}
		csid = 64 + uint8(b[0])
	} else if csid == 1 {
		// 3 byte form
		b := make([]byte, 2)
		if _, err := io.ReadFull(stream.conn, b); err != nil {
			return err
		}
		csid = 64 + uint8(b[0]) + (uint8(b[1]) << 8)
	}

	// Read message header based on format
	var timestamp uint32
	var messageLength uint32
	var messageType uint8
	var messageStreamID uint32

	switch fmt {
	case 0:
		// Type 0: Full header (11 bytes)
		header := make([]byte, 11)
		if _, err := io.ReadFull(stream.conn, header); err != nil {
			return err
		}
		timestamp = uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])
		messageLength = uint32(header[3])<<16 | uint32(header[4])<<8 | uint32(header[5])
		messageType = header[6]
		messageStreamID = binary.LittleEndian.Uint32(header[7:11])
		_ = messageStreamID // Used for full message handling

	case 1:
		// Type 1: Same stream ID (7 bytes)
		header := make([]byte, 7)
		if _, err := io.ReadFull(stream.conn, header); err != nil {
			return err
		}
		timestamp = uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])
		messageLength = uint32(header[3])<<16 | uint32(header[4])<<8 | uint32(header[5])
		messageType = header[6]

	case 2:
		// Type 2: Same length and stream ID (3 bytes)
		header := make([]byte, 3)
		if _, err := io.ReadFull(stream.conn, header); err != nil {
			return err
		}
		timestamp = uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])

	case 3:
		// Type 3: No header
		// Use previous values
	}

	// Read extended timestamp if needed
	if timestamp == 0xffffff {
		extTime := make([]byte, 4)
		if _, err := io.ReadFull(stream.conn, extTime); err != nil {
			return err
		}
		timestamp = binary.BigEndian.Uint32(extTime)
	}

	// Read message payload
	payload := make([]byte, messageLength)
	if _, err := io.ReadFull(stream.conn, payload); err != nil {
		return err
	}

	// Update statistics
	stream.bytesReceived += int64(messageLength)

	// Handle message based on type
	switch messageType {
	case RTMP_MSG_CommandAMF0:
		return r.handleCommand(stream, payload)
	case RTMP_MSG_Audio:
		return r.handleAudioData(stream, payload)
	case RTMP_MSG_Video:
		return r.handleVideoData(stream, payload)
	case RTMP_MSG_DataAMF0:
		return r.handleMetadata(stream, payload)
	}

	return nil
}

// handleCommand handles RTMP command messages
func (r *RTMPServer) handleCommand(stream *RTMPStream, payload []byte) error {
	// Parse AMF0 command
	reader := bytes.NewReader(payload)

	// Read command name
	cmdName, err := r.readAMF0String(reader)
	if err != nil {
		return err
	}

	switch cmdName {
	case "connect":
		return r.handleConnect(stream, reader)
	case "releaseStream":
		return r.handleReleaseStream(stream, reader)
	case "FCPublish":
		return r.handleFCPublish(stream, reader)
	case "createStream":
		return r.handleCreateStream(stream, reader)
	case "publish":
		return r.handlePublish(stream, reader)
	case "play":
		return r.handlePlay(stream, reader)
	}

	return nil
}

// handleConnect handles connect command
func (r *RTMPServer) handleConnect(stream *RTMPStream, reader *bytes.Reader) error {
	// Send connect response
	response := r.createConnectResponse()
	return r.sendMessage(stream.conn, RTMP_MSG_CommandAMF0, response)
}

// handlePublish handles publish command
func (r *RTMPServer) handlePublish(stream *RTMPStream, reader *bytes.Reader) error {
	// Read transaction ID
	_, _ = r.readAMF0Number(reader)

	// Skip null
	reader.Seek(1, io.SeekCurrent)

	// Read stream name
	streamName, err := r.readAMF0String(reader)
	if err != nil {
		return err
	}

	// Read publish type
	publishType, _ := r.readAMF0String(reader)
	_ = publishType // Used for different publish modes

	// Extract stream key from name
	stream.streamKey = streamName
	stream.mountPoint = "/" + streamName

	// Verify stream key
	userID, err := r.verifyStreamKey(stream.streamKey)
	if err != nil {
		return r.sendStatusMessage(stream.conn, "error", "Invalid stream key")
	}

	stream.userID = userID

	// Add to active streams
	r.mu.Lock()
	r.streams[stream.streamKey] = stream
	r.mu.Unlock()

	// Update database
	r.db.Exec(`
		INSERT INTO broadcasts (mount_point, type, name, user_id, stream_key, is_active, started_at)
		VALUES (?, 'live', ?, ?, ?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(mount_point) DO UPDATE SET
			is_active = 1,
			started_at = CURRENT_TIMESTAMP
	`, stream.mountPoint, streamName, userID, stream.streamKey)

	// Send publish start response
	return r.sendStatusMessage(stream.conn, "status", "Publishing started")
}

// handleAudioData handles audio data
func (r *RTMPServer) handleAudioData(stream *RTMPStream, payload []byte) error {
	// Process audio data
	// In production, this would be sent to transcoder or saved

	// Update stream info if first audio packet
	if stream.audioCodec == "" && len(payload) > 0 {
		audioFormat := payload[0] >> 4
		switch audioFormat {
		case 10:
			stream.audioCodec = "aac"
		case 2:
			stream.audioCodec = "mp3"
		case 11:
			stream.audioCodec = "speex"
		}
	}

	return nil
}

// handleVideoData handles video data
func (r *RTMPServer) handleVideoData(stream *RTMPStream, payload []byte) error {
	// Process video data
	// In production, this would be sent to transcoder or saved

	// Update stream info if first video packet
	if stream.videoCodec == "" && len(payload) > 0 {
		codecID := payload[0] & 0x0f
		switch codecID {
		case 7:
			stream.videoCodec = "h264"
		case 12:
			stream.videoCodec = "h265"
		case 2:
			stream.videoCodec = "h263"
		}
	}

	return nil
}

// handleMetadata handles metadata
func (r *RTMPServer) handleMetadata(stream *RTMPStream, payload []byte) error {
	// Parse metadata
	reader := bytes.NewReader(payload)

	// Read @setDataFrame
	eventType, _ := r.readAMF0String(reader)
	if eventType != "@setDataFrame" && eventType != "onMetaData" {
		return nil
	}

	// Parse metadata object
	// In production, extract video dimensions, bitrate, etc.

	return nil
}

// verifyStreamKey verifies a stream key and returns user ID
func (r *RTMPServer) verifyStreamKey(streamKey string) (int, error) {
	var userID int
	var mountPoint string

	err := r.db.QueryRow(`
		SELECT user_id, mount_point FROM broadcasts
		WHERE stream_key = ? AND is_enabled = 1
	`, streamKey).Scan(&userID, &mountPoint)

	if err != nil {
		// Try API token
		err = r.db.QueryRow(`
			SELECT user_id FROM api_tokens
			WHERE token = ? AND is_active = 1
		`, streamKey).Scan(&userID)

		if err != nil {
			return 0, fmt.Errorf("invalid stream key")
		}
	}

	return userID, nil
}

// removeStream removes a stream from active streams
func (r *RTMPServer) removeStream(stream *RTMPStream) {
	stream.isActive = false
	stream.conn.Close()

	r.mu.Lock()
	delete(r.streams, stream.streamKey)
	r.mu.Unlock()

	// Update database
	r.db.Exec(`
		UPDATE broadcasts
		SET is_active = 0, stopped_at = CURRENT_TIMESTAMP,
			listeners_peak = ?
		WHERE mount_point = ?
	`, stream.viewers, stream.mountPoint)

	// Record history
	duration := time.Since(stream.startTime)
	_ = duration // Will be used for detailed statistics
	r.db.Exec(`
		INSERT INTO broadcast_history (
			broadcast_id, started_at, stopped_at,
			peak_listeners, total_bytes_sent
		)
		SELECT id, ?, CURRENT_TIMESTAMP, ?, ?
		FROM broadcasts WHERE mount_point = ?
	`, stream.startTime, stream.viewers, stream.bytesReceived, stream.mountPoint)
}

// Close closes a stream
func (s *RTMPStream) Close() {
	s.isActive = false
	if s.conn != nil {
		s.conn.Close()
	}
}

// AMF0 helper functions

func (r *RTMPServer) readAMF0String(reader *bytes.Reader) (string, error) {
	// Read type marker
	marker, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	if marker != 0x02 { // String marker
		return "", fmt.Errorf("expected string marker, got %x", marker)
	}

	// Read length
	lengthBytes := make([]byte, 2)
	if _, err := reader.Read(lengthBytes); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(lengthBytes)

	// Read string
	strBytes := make([]byte, length)
	if _, err := reader.Read(strBytes); err != nil {
		return "", err
	}

	return string(strBytes), nil
}

func (r *RTMPServer) readAMF0Number(reader *bytes.Reader) (float64, error) {
	// Read type marker
	marker, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	if marker != 0x00 { // Number marker
		return 0, fmt.Errorf("expected number marker, got %x", marker)
	}

	// Read 8 bytes as float64
	var num float64
	if err := binary.Read(reader, binary.BigEndian, &num); err != nil {
		return 0, err
	}

	return num, nil
}

func (r *RTMPServer) writeAMF0String(buffer *bytes.Buffer, str string) {
	buffer.WriteByte(0x02) // String marker
	binary.Write(buffer, binary.BigEndian, uint16(len(str)))
	buffer.WriteString(str)
}

func (r *RTMPServer) writeAMF0Number(buffer *bytes.Buffer, num float64) {
	buffer.WriteByte(0x00) // Number marker
	binary.Write(buffer, binary.BigEndian, num)
}

func (r *RTMPServer) writeAMF0Object(buffer *bytes.Buffer) {
	buffer.WriteByte(0x03) // Object marker
}

func (r *RTMPServer) writeAMF0ObjectEnd(buffer *bytes.Buffer) {
	buffer.Write([]byte{0x00, 0x00, 0x09}) // Object end marker
}

func (r *RTMPServer) writeAMF0Null(buffer *bytes.Buffer) {
	buffer.WriteByte(0x05) // Null marker
}

// Helper functions for sending messages

func (r *RTMPServer) createConnectResponse() []byte {
	var buf bytes.Buffer

	// _result
	r.writeAMF0String(&buf, "_result")

	// Transaction ID
	r.writeAMF0Number(&buf, 1)

	// Properties object
	r.writeAMF0Object(&buf)
	buf.WriteString("fmsVer")
	r.writeAMF0String(&buf, "CASRAD/1.0")
	buf.WriteString("capabilities")
	r.writeAMF0Number(&buf, 31)
	r.writeAMF0ObjectEnd(&buf)

	// Information object
	r.writeAMF0Object(&buf)
	buf.WriteString("level")
	r.writeAMF0String(&buf, "status")
	buf.WriteString("code")
	r.writeAMF0String(&buf, "NetConnection.Connect.Success")
	buf.WriteString("description")
	r.writeAMF0String(&buf, "Connection succeeded")
	r.writeAMF0ObjectEnd(&buf)

	return buf.Bytes()
}

func (r *RTMPServer) sendMessage(conn net.Conn, messageType uint8, payload []byte) error {
	// Simplified message sending
	// In production, this would handle chunking properly

	var buf bytes.Buffer

	// Basic header (fmt=0, csid=3)
	buf.WriteByte(0x03)

	// Message header
	// Timestamp (3 bytes)
	buf.Write([]byte{0x00, 0x00, 0x00})

	// Message length (3 bytes)
	length := len(payload)
	buf.WriteByte(byte(length >> 16))
	buf.WriteByte(byte(length >> 8))
	buf.WriteByte(byte(length))

	// Message type
	buf.WriteByte(messageType)

	// Message stream ID (little-endian)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})

	// Payload
	buf.Write(payload)

	_, err := conn.Write(buf.Bytes())
	return err
}

func (r *RTMPServer) sendStatusMessage(conn net.Conn, level, description string) error {
	var buf bytes.Buffer

	r.writeAMF0String(&buf, "onStatus")
	r.writeAMF0Number(&buf, 0)
	r.writeAMF0Null(&buf)

	r.writeAMF0Object(&buf)
	buf.WriteString("level")
	r.writeAMF0String(&buf, level)
	buf.WriteString("description")
	r.writeAMF0String(&buf, description)
	r.writeAMF0ObjectEnd(&buf)

	return r.sendMessage(conn, RTMP_MSG_CommandAMF0, buf.Bytes())
}

// Additional stub handlers
func (r *RTMPServer) handleReleaseStream(stream *RTMPStream, reader *bytes.Reader) error {
	return nil
}

func (r *RTMPServer) handleFCPublish(stream *RTMPStream, reader *bytes.Reader) error {
	return nil
}

func (r *RTMPServer) handleCreateStream(stream *RTMPStream, reader *bytes.Reader) error {
	// Send createStream response
	var buf bytes.Buffer
	r.writeAMF0String(&buf, "_result")
	r.writeAMF0Number(&buf, 4) // Transaction ID
	r.writeAMF0Null(&buf)
	r.writeAMF0Number(&buf, 1) // Stream ID

	return r.sendMessage(stream.conn, RTMP_MSG_CommandAMF0, buf.Bytes())
}

func (r *RTMPServer) handlePlay(stream *RTMPStream, reader *bytes.Reader) error {
	// Handle play request for viewers
	return r.sendStatusMessage(stream.conn, "status", "Play started")
}

// GetActiveStreams returns all active streams
func (r *RTMPServer) GetActiveStreams() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	streams := []map[string]interface{}{}
	for key, stream := range r.streams {
		streams = append(streams, map[string]interface{}{
			"stream_key":    key,
			"mount_point":   stream.mountPoint,
			"user_id":       stream.userID,
			"video_codec":   stream.videoCodec,
			"audio_codec":   stream.audioCodec,
			"viewers":       stream.viewers,
			"bytes_received": stream.bytesReceived,
			"duration":      time.Since(stream.startTime).Seconds(),
			"is_active":     stream.isActive,
		})
	}

	return streams
}