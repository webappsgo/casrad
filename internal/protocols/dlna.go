package protocols

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/gorilla/mux"
)

// DLNAServer implements a DLNA/UPnP media server
type DLNAServer struct {
	db       *database.Engine
	ssdpPort int
	httpPort int
	uuid     string
	name     string
	enabled  bool
	listener net.PacketConn
	router   *mux.Router
}

// NewDLNAServer creates a new DLNA server
func NewDLNAServer(db *database.Engine, httpPort int) *DLNAServer {
	return &DLNAServer{
		db:       db,
		ssdpPort: 1900,
		httpPort: httpPort,
		uuid:     "uuid:casrad-dlna-server-001",
		name:     "CASRAD Media Server",
		enabled:  true,
		router:   mux.NewRouter(),
	}
}

// Start starts the DLNA server
func (d *DLNAServer) Start() error {
	if !d.enabled {
		return nil
	}

	// Setup HTTP routes
	d.setupRoutes()

	// Start SSDP discovery
	if err := d.startSSDP(); err != nil {
		return fmt.Errorf("failed to start SSDP: %w", err)
	}

	log.Printf("DLNA server started on port %d", d.httpPort)
	return nil
}

// Stop stops the DLNA server
func (d *DLNAServer) Stop() error {
	if d.listener != nil {
		d.listener.Close()
	}
	return nil
}

// setupRoutes configures HTTP routes for DLNA
func (d *DLNAServer) setupRoutes() {
	// Device description
	d.router.HandleFunc("/dlna/device.xml", d.handleDeviceDescription)
	
	// Service descriptions
	d.router.HandleFunc("/dlna/ContentDirectory.xml", d.handleContentDirectoryDescription)
	d.router.HandleFunc("/dlna/ConnectionManager.xml", d.handleConnectionManagerDescription)
	d.router.HandleFunc("/dlna/MediaReceiverRegistrar.xml", d.handleMediaReceiverDescription)
	
	// Control endpoints
	d.router.HandleFunc("/dlna/control/ContentDirectory", d.handleContentDirectoryControl).Methods("POST")
	d.router.HandleFunc("/dlna/control/ConnectionManager", d.handleConnectionManagerControl).Methods("POST")
	d.router.HandleFunc("/dlna/control/MediaReceiverRegistrar", d.handleMediaReceiverControl).Methods("POST")
	
	// Media streaming
	d.router.HandleFunc("/dlna/media/{id}", d.handleMediaStream).Methods("GET", "HEAD")
	d.router.HandleFunc("/dlna/albumart/{id}", d.handleAlbumArt).Methods("GET")
}

// ServeHTTP implements http.Handler
func (d *DLNAServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.router.ServeHTTP(w, r)
}

// startSSDP starts the SSDP discovery service
func (d *DLNAServer) startSSDP() error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", d.ssdpPort))
	if err != nil {
		return err
	}

	listener, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}

	d.listener = listener

	// Set buffer size
	listener.SetReadBuffer(1048576)

	// Note: Multicast group joining would require using golang.org/x/net/ipv4
	// For now, we'll just listen on the port

	// Start SSDP listener
	go d.ssdpListener()

	// Send initial announcements
	go d.sendAnnouncements()

	return nil
}

// ssdpListener handles SSDP discovery requests
func (d *DLNAServer) ssdpListener() {
	buffer := make([]byte, 2048)

	for {
		n, addr, err := d.listener.ReadFrom(buffer)
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				log.Printf("SSDP read error: %v", err)
			}
			break
		}

		msg := string(buffer[:n])
		if strings.HasPrefix(msg, "M-SEARCH") {
			go d.handleMSearch(msg, addr)
		}
	}
}

// handleMSearch handles SSDP M-SEARCH discovery requests
func (d *DLNAServer) handleMSearch(msg string, addr net.Addr) {
	// Parse M-SEARCH request
	if !strings.Contains(msg, "ssdp:discover") {
		return
	}

	// Check for supported search targets
	supportedTargets := []string{
		"ssdp:all",
		"upnp:rootdevice",
		"urn:schemas-upnp-org:device:MediaServer:1",
		"urn:schemas-upnp-org:service:ContentDirectory:1",
		d.uuid,
	}

	var targetFound bool
	for _, target := range supportedTargets {
		if strings.Contains(msg, target) {
			targetFound = true
			break
		}
	}

	if !targetFound {
		return
	}

	// Send response
	location := fmt.Sprintf("http://%s:%d/dlna/device.xml", d.getLocalIP(), d.httpPort)
	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"CACHE-CONTROL: max-age=1800\r\n"+
			"EXT:\r\n"+
			"LOCATION: %s\r\n"+
			"SERVER: CASRAD/1.0 UPnP/1.0 DLNA/1.5\r\n"+
			"ST: urn:schemas-upnp-org:device:MediaServer:1\r\n"+
			"USN: %s::urn:schemas-upnp-org:device:MediaServer:1\r\n"+
			"\r\n",
		location, d.uuid)

	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		if conn, ok := d.listener.(*net.UDPConn); ok {
			conn.WriteToUDP([]byte(response), udpAddr)
		}
	}
}

// sendAnnouncements sends SSDP announcements
func (d *DLNAServer) sendAnnouncements() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		d.sendAliveMessage()
		<-ticker.C
	}
}

// sendAliveMessage sends SSDP alive announcement
func (d *DLNAServer) sendAliveMessage() {
	location := fmt.Sprintf("http://%s:%d/dlna/device.xml", d.getLocalIP(), d.httpPort)
	message := fmt.Sprintf(
		"NOTIFY * HTTP/1.1\r\n"+
			"HOST: 239.255.255.250:1900\r\n"+
			"CACHE-CONTROL: max-age=1800\r\n"+
			"LOCATION: %s\r\n"+
			"NT: upnp:rootdevice\r\n"+
			"NTS: ssdp:alive\r\n"+
			"SERVER: CASRAD/1.0 UPnP/1.0 DLNA/1.5\r\n"+
			"USN: %s::upnp:rootdevice\r\n"+
			"\r\n",
		location, d.uuid)

	multicastAddr, _ := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if conn, ok := d.listener.(*net.UDPConn); ok {
		conn.WriteToUDP([]byte(message), multicastAddr)
	}
}

// getLocalIP returns the local IP address
func (d *DLNAServer) getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

// handleDeviceDescription returns the device description XML
func (d *DLNAServer) handleDeviceDescription(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0" xmlns:dlna="urn:schemas-dlna-org:device-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <device>
    <deviceType>urn:schemas-upnp-org:device:MediaServer:1</deviceType>
    <friendlyName>%s</friendlyName>
    <manufacturer>CASRAD</manufacturer>
    <manufacturerURL>https://github.com/casapps/casrad</manufacturerURL>
    <modelDescription>CASRAD DLNA Media Server</modelDescription>
    <modelName>CASRAD</modelName>
    <modelNumber>1.0</modelNumber>
    <modelURL>https://github.com/casapps/casrad</modelURL>
    <serialNumber>00000001</serialNumber>
    <UDN>%s</UDN>
    <dlna:X_DLNACAP/>
    <dlna:X_DLNADOC>DMS-1.50</dlna:X_DLNADOC>
    <serviceList>
      <service>
        <serviceType>urn:schemas-upnp-org:service:ContentDirectory:1</serviceType>
        <serviceId>urn:upnp-org:serviceId:ContentDirectory</serviceId>
        <SCPDURL>/dlna/ContentDirectory.xml</SCPDURL>
        <controlURL>/dlna/control/ContentDirectory</controlURL>
        <eventSubURL>/dlna/event/ContentDirectory</eventSubURL>
      </service>
      <service>
        <serviceType>urn:schemas-upnp-org:service:ConnectionManager:1</serviceType>
        <serviceId>urn:upnp-org:serviceId:ConnectionManager</serviceId>
        <SCPDURL>/dlna/ConnectionManager.xml</SCPDURL>
        <controlURL>/dlna/control/ConnectionManager</controlURL>
        <eventSubURL>/dlna/event/ConnectionManager</eventSubURL>
      </service>
      <service>
        <serviceType>urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1</serviceType>
        <serviceId>urn:microsoft.com:serviceId:X_MS_MediaReceiverRegistrar</serviceId>
        <SCPDURL>/dlna/MediaReceiverRegistrar.xml</SCPDURL>
        <controlURL>/dlna/control/MediaReceiverRegistrar</controlURL>
        <eventSubURL>/dlna/event/MediaReceiverRegistrar</eventSubURL>
      </service>
    </serviceList>
  </device>
</root>`, d.name, d.uuid)

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml))
}

// handleContentDirectoryDescription returns the ContentDirectory service description
func (d *DLNAServer) handleContentDirectoryDescription(w http.ResponseWriter, r *http.Request) {
	xml := `<?xml version="1.0"?>
<scpd xmlns="urn:schemas-upnp-org:service-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <actionList>
    <action>
      <name>Browse</name>
      <argumentList>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>BrowseFlag</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_BrowseFlag</relatedStateVariable>
        </argument>
        <argument>
          <name>Filter</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Filter</relatedStateVariable>
        </argument>
        <argument>
          <name>StartingIndex</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Index</relatedStateVariable>
        </argument>
        <argument>
          <name>RequestedCount</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>SortCriteria</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_SortCriteria</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
        <argument>
          <name>NumberReturned</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>TotalMatches</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>UpdateID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_UpdateID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSearchCapabilities</name>
      <argumentList>
        <argument>
          <name>SearchCaps</name>
          <direction>out</direction>
          <relatedStateVariable>SearchCapabilities</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSortCapabilities</name>
      <argumentList>
        <argument>
          <name>SortCaps</name>
          <direction>out</direction>
          <relatedStateVariable>SortCapabilities</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSystemUpdateID</name>
      <argumentList>
        <argument>
          <name>Id</name>
          <direction>out</direction>
          <relatedStateVariable>SystemUpdateID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
  </actionList>
  <serviceStateTable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ObjectID</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_BrowseFlag</name>
      <dataType>string</dataType>
      <allowedValueList>
        <allowedValue>BrowseMetadata</allowedValue>
        <allowedValue>BrowseDirectChildren</allowedValue>
      </allowedValueList>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Filter</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Index</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Count</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_SortCriteria</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Result</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_UpdateID</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>SystemUpdateID</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>SearchCapabilities</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>SortCapabilities</name>
      <dataType>string</dataType>
    </stateVariable>
  </serviceStateTable>
</scpd>`

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml))
}

// handleConnectionManagerDescription returns the ConnectionManager service description
func (d *DLNAServer) handleConnectionManagerDescription(w http.ResponseWriter, r *http.Request) {
	xml := `<?xml version="1.0"?>
<scpd xmlns="urn:schemas-upnp-org:service-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <actionList>
    <action>
      <name>GetProtocolInfo</name>
      <argumentList>
        <argument>
          <name>Source</name>
          <direction>out</direction>
          <relatedStateVariable>SourceProtocolInfo</relatedStateVariable>
        </argument>
        <argument>
          <name>Sink</name>
          <direction>out</direction>
          <relatedStateVariable>SinkProtocolInfo</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetCurrentConnectionIDs</name>
      <argumentList>
        <argument>
          <name>ConnectionIDs</name>
          <direction>out</direction>
          <relatedStateVariable>CurrentConnectionIDs</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetCurrentConnectionInfo</name>
      <argumentList>
        <argument>
          <name>ConnectionID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ConnectionID</relatedStateVariable>
        </argument>
        <argument>
          <name>RcsID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_RcsID</relatedStateVariable>
        </argument>
        <argument>
          <name>AVTransportID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_AVTransportID</relatedStateVariable>
        </argument>
        <argument>
          <name>ProtocolInfo</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ProtocolInfo</relatedStateVariable>
        </argument>
        <argument>
          <name>PeerConnectionManager</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ConnectionManager</relatedStateVariable>
        </argument>
        <argument>
          <name>PeerConnectionID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ConnectionID</relatedStateVariable>
        </argument>
        <argument>
          <name>Direction</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Direction</relatedStateVariable>
        </argument>
        <argument>
          <name>Status</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ConnectionStatus</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
  </actionList>
  <serviceStateTable>
    <stateVariable sendEvents="yes">
      <name>SourceProtocolInfo</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>SinkProtocolInfo</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>CurrentConnectionIDs</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ConnectionID</name>
      <dataType>i4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_RcsID</name>
      <dataType>i4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_AVTransportID</name>
      <dataType>i4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ProtocolInfo</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ConnectionManager</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Direction</name>
      <dataType>string</dataType>
      <allowedValueList>
        <allowedValue>Input</allowedValue>
        <allowedValue>Output</allowedValue>
      </allowedValueList>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ConnectionStatus</name>
      <dataType>string</dataType>
      <allowedValueList>
        <allowedValue>OK</allowedValue>
        <allowedValue>ContentFormatMismatch</allowedValue>
        <allowedValue>InsufficientBandwidth</allowedValue>
        <allowedValue>UnreliableChannel</allowedValue>
        <allowedValue>Unknown</allowedValue>
      </allowedValueList>
    </stateVariable>
  </serviceStateTable>
</scpd>`

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml))
}

// handleMediaReceiverDescription returns the MediaReceiverRegistrar service description
func (d *DLNAServer) handleMediaReceiverDescription(w http.ResponseWriter, r *http.Request) {
	xml := `<?xml version="1.0"?>
<scpd xmlns="urn:schemas-upnp-org:service-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <actionList>
    <action>
      <name>IsAuthorized</name>
      <argumentList>
        <argument>
          <name>DeviceID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_DeviceID</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>RegisterDevice</name>
      <argumentList>
        <argument>
          <name>RegistrationReqMsg</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_RegistrationReqMsg</relatedStateVariable>
        </argument>
        <argument>
          <name>RegistrationRespMsg</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_RegistrationRespMsg</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>IsValidated</name>
      <argumentList>
        <argument>
          <name>DeviceID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_DeviceID</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
  </actionList>
  <serviceStateTable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_DeviceID</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Result</name>
      <dataType>i4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_RegistrationReqMsg</name>
      <dataType>bin.base64</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_RegistrationRespMsg</name>
      <dataType>bin.base64</dataType>
    </stateVariable>
  </serviceStateTable>
</scpd>`

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml))
}

// handleContentDirectoryControl handles ContentDirectory control requests
func (d *DLNAServer) handleContentDirectoryControl(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)
	defer r.Body.Close()

	bodyStr := string(body)
	soapAction := r.Header.Get("SOAPACTION")

	var response string

	if strings.Contains(soapAction, "Browse") {
		response = d.handleBrowse(bodyStr)
	} else if strings.Contains(soapAction, "GetSearchCapabilities") {
		response = d.handleGetSearchCapabilities()
	} else if strings.Contains(soapAction, "GetSortCapabilities") {
		response = d.handleGetSortCapabilities()
	} else if strings.Contains(soapAction, "GetSystemUpdateID") {
		response = d.handleGetSystemUpdateID()
	} else {
		http.Error(w, "Action not implemented", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(response))
}

// handleBrowse handles Browse requests
func (d *DLNAServer) handleBrowse(body string) string {
	// Parse ObjectID from request
	objectID := "0" // Default to root
	if idx := strings.Index(body, "<ObjectID>"); idx >= 0 {
		end := strings.Index(body[idx:], "</ObjectID>")
		if end > 0 {
			objectID = body[idx+10 : idx+end]
		}
	}

	// Parse StartingIndex
	startingIndex := 0
	if idx := strings.Index(body, "<StartingIndex>"); idx >= 0 {
		end := strings.Index(body[idx:], "</StartingIndex>")
		if end > 0 {
			startingIndex, _ = strconv.Atoi(body[idx+15 : idx+end])
		}
	}

	// Parse RequestedCount
	requestedCount := 100
	if idx := strings.Index(body, "<RequestedCount>"); idx >= 0 {
		end := strings.Index(body[idx:], "</RequestedCount>")
		if end > 0 {
			requestedCount, _ = strconv.Atoi(body[idx+16 : idx+end])
		}
	}

	// Build DIDL response
	var didl string
	var numberReturned, totalMatches int

	if objectID == "0" {
		// Root container - return top level containers
		didl = `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">
`
		didl += `<container id="1" parentID="0" restricted="1"><dc:title>Music</dc:title><upnp:class>object.container.storageFolder</upnp:class></container>
`
		didl += `<container id="2" parentID="0" restricted="1"><dc:title>Playlists</dc:title><upnp:class>object.container.storageFolder</upnp:class></container>
`
		didl += `<container id="3" parentID="0" restricted="1"><dc:title>Artists</dc:title><upnp:class>object.container.person.musicArtist</upnp:class></container>
`
		didl += `<container id="4" parentID="0" restricted="1"><dc:title>Albums</dc:title><upnp:class>object.container.album.musicAlbum</upnp:class></container>
`
		didl += `<container id="5" parentID="0" restricted="1"><dc:title>Genres</dc:title><upnp:class>object.container.genre.musicGenre</upnp:class></container>
`
		didl += `</DIDL-Lite>`
		numberReturned = 5
		totalMatches = 5
	} else if objectID == "1" {
		// Music container - return tracks
		rows, err := d.db.Query(`
			SELECT id, title, artist, album, duration, file_path
			FROM tracks
			ORDER BY title
			LIMIT ? OFFSET ?
		`, requestedCount, startingIndex)

		if err == nil {
			defer rows.Close()

			didl = `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">
`

			for rows.Next() {
				var id int
				var title, artist, album, filePath string
				var duration int

				rows.Scan(&id, &title, &artist, &album, &duration, &filePath)

				resURL := fmt.Sprintf("http://%s:%d/dlna/media/%d", d.getLocalIP(), d.httpPort, id)
				didl += fmt.Sprintf(
					`<item id="track_%d" parentID="1" restricted="1">`+
						`<dc:title>%s</dc:title>`+
						`<upnp:class>object.item.audioItem.musicTrack</upnp:class>`+
						`<upnp:artist>%s</upnp:artist>`+
						`<upnp:album>%s</upnp:album>`+
						`<res protocolInfo="http-get:*:audio/mpeg:DLNA.ORG_PN=MP3;DLNA.ORG_OP=01;DLNA.ORG_CI=0">%s</res>`+
						`</item>
`,
					id, escapeXML(title), escapeXML(artist), escapeXML(album), resURL)
				numberReturned++
			}

			didl += `</DIDL-Lite>`

			// Get total count
			d.db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&totalMatches)
		}
	} else {
		// Empty container for now
		didl = `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"></DIDL-Lite>`
	}

	// Escape DIDL for SOAP
	didl = escapeXML(didl)

	return fmt.Sprintf(
		`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:BrowseResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1">
      <Result>%s</Result>
      <NumberReturned>%d</NumberReturned>
      <TotalMatches>%d</TotalMatches>
      <UpdateID>1</UpdateID>
    </u:BrowseResponse>
  </s:Body>
</s:Envelope>`,
		didl, numberReturned, totalMatches)
}

// handleGetSearchCapabilities returns search capabilities
func (d *DLNAServer) handleGetSearchCapabilities() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetSearchCapabilitiesResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1">
      <SearchCaps>dc:title,upnp:artist,upnp:album</SearchCaps>
    </u:GetSearchCapabilitiesResponse>
  </s:Body>
</s:Envelope>`
}

// handleGetSortCapabilities returns sort capabilities
func (d *DLNAServer) handleGetSortCapabilities() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetSortCapabilitiesResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1">
      <SortCaps>dc:title,upnp:artist,upnp:album</SortCaps>
    </u:GetSortCapabilitiesResponse>
  </s:Body>
</s:Envelope>`
}

// handleGetSystemUpdateID returns system update ID
func (d *DLNAServer) handleGetSystemUpdateID() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetSystemUpdateIDResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1">
      <Id>1</Id>
    </u:GetSystemUpdateIDResponse>
  </s:Body>
</s:Envelope>`
}

// handleConnectionManagerControl handles ConnectionManager control requests
func (d *DLNAServer) handleConnectionManagerControl(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)
	defer r.Body.Close()

	soapAction := r.Header.Get("SOAPACTION")

	var response string

	if strings.Contains(soapAction, "GetProtocolInfo") {
		response = d.handleGetProtocolInfo()
	} else if strings.Contains(soapAction, "GetCurrentConnectionIDs") {
		response = d.handleGetCurrentConnectionIDs()
	} else if strings.Contains(soapAction, "GetCurrentConnectionInfo") {
		response = d.handleGetCurrentConnectionInfo()
	} else {
		http.Error(w, "Action not implemented", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(response))
}

// handleGetProtocolInfo returns protocol info
func (d *DLNAServer) handleGetProtocolInfo() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetProtocolInfoResponse xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1">
      <Source>http-get:*:audio/mpeg:DLNA.ORG_PN=MP3,http-get:*:audio/mp4:DLNA.ORG_PN=AAC_ISO,http-get:*:audio/x-flac:*</Source>
      <Sink></Sink>
    </u:GetProtocolInfoResponse>
  </s:Body>
</s:Envelope>`
}

// handleGetCurrentConnectionIDs returns current connection IDs
func (d *DLNAServer) handleGetCurrentConnectionIDs() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetCurrentConnectionIDsResponse xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1">
      <ConnectionIDs>0</ConnectionIDs>
    </u:GetCurrentConnectionIDsResponse>
  </s:Body>
</s:Envelope>`
}

// handleGetCurrentConnectionInfo returns current connection info
func (d *DLNAServer) handleGetCurrentConnectionInfo() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetCurrentConnectionInfoResponse xmlns:u="urn:schemas-upnp-org:service:ConnectionManager:1">
      <RcsID>-1</RcsID>
      <AVTransportID>-1</AVTransportID>
      <ProtocolInfo></ProtocolInfo>
      <PeerConnectionManager></PeerConnectionManager>
      <PeerConnectionID>-1</PeerConnectionID>
      <Direction>Output</Direction>
      <Status>OK</Status>
    </u:GetCurrentConnectionInfoResponse>
  </s:Body>
</s:Envelope>`
}

// handleMediaReceiverControl handles MediaReceiverRegistrar control requests
func (d *DLNAServer) handleMediaReceiverControl(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)
	defer r.Body.Close()

	soapAction := r.Header.Get("SOAPACTION")

	var response string

	if strings.Contains(soapAction, "IsAuthorized") {
		response = d.handleIsAuthorized()
	} else if strings.Contains(soapAction, "IsValidated") {
		response = d.handleIsValidated()
	} else if strings.Contains(soapAction, "RegisterDevice") {
		response = d.handleRegisterDevice()
	} else {
		http.Error(w, "Action not implemented", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(response))
}

// handleIsAuthorized returns authorization status
func (d *DLNAServer) handleIsAuthorized() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:IsAuthorizedResponse xmlns:u="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1">
      <Result>1</Result>
    </u:IsAuthorizedResponse>
  </s:Body>
</s:Envelope>`
}

// handleIsValidated returns validation status
func (d *DLNAServer) handleIsValidated() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:IsValidatedResponse xmlns:u="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1">
      <Result>1</Result>
    </u:IsValidatedResponse>
  </s:Body>
</s:Envelope>`
}

// handleRegisterDevice registers a device
func (d *DLNAServer) handleRegisterDevice() string {
	return `<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:RegisterDeviceResponse xmlns:u="urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1">
      <RegistrationRespMsg></RegistrationRespMsg>
    </u:RegisterDeviceResponse>
  </s:Body>
</s:Envelope>`
}

// handleMediaStream streams media files
func (d *DLNAServer) handleMediaStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid media ID", http.StatusBadRequest)
		return
	}

	// Get track info
	var filePath, contentType string
	var fileSize int64
	err = d.db.QueryRow(`
		SELECT file_path, file_size, file_type
		FROM tracks
		WHERE id = ?
	`, id).Scan(&filePath, &fileSize, &contentType)

	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Map file type to content type
	switch contentType {
	case "mp3":
		contentType = "audio/mpeg"
	case "flac":
		contentType = "audio/x-flac"
	case "m4a", "aac":
		contentType = "audio/mp4"
	case "ogg":
		contentType = "audio/ogg"
	default:
		contentType = "audio/mpeg"
	}

	// Serve the file
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", "DLNA.ORG_PN=MP3;DLNA.ORG_OP=01;DLNA.ORG_CI=0")

	http.ServeFile(w, r, filePath)
}

// handleAlbumArt serves album artwork
func (d *DLNAServer) handleAlbumArt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid album ID", http.StatusBadRequest)
		return
	}

	// Get album art path
	var coverPath string
	err = d.db.QueryRow(`
		SELECT cover_art_path
		FROM albums
		WHERE id = ?
	`, id).Scan(&coverPath)

	if err != nil || coverPath == "" {
		// Return default image or 404
		http.Error(w, "Album art not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, coverPath)
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// Enable enables the DLNA server
func (d *DLNAServer) Enable() {
	d.enabled = true
	d.db.SetSetting("dlna.enabled", "true", nil)
}

// Disable disables the DLNA server
func (d *DLNAServer) Disable() {
	d.enabled = false
	d.db.SetSetting("dlna.enabled", "false", nil)
	d.Stop()
}