// Package service - Notification service for WebUI notifications
// See AI.md PART 18 for notification specification
package service

import (
	"sync"
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotifySuccess  NotificationType = "success"
	NotifyInfo     NotificationType = "info"
	NotifyWarning  NotificationType = "warning"
	NotifyError    NotificationType = "error"
	NotifySecurity NotificationType = "security"
)

// NotificationIcon returns the icon for a notification type
func (t NotificationType) Icon() string {
	switch t {
	case NotifySuccess:
		return "check-circle" // Tabler icon name
	case NotifyInfo:
		return "info-circle"
	case NotifyWarning:
		return "alert-triangle"
	case NotifyError:
		return "x-circle"
	case NotifySecurity:
		return "lock"
	default:
		return "bell"
	}
}

// AutoDismiss returns the auto-dismiss duration for a notification type
func (t NotificationType) AutoDismiss() time.Duration {
	switch t {
	case NotifySuccess, NotifyInfo:
		return 5 * time.Second
	case NotifyWarning:
		return 10 * time.Second
	case NotifyError, NotifySecurity:
		return 0 // Manual dismiss required
	default:
		return 5 * time.Second
	}
}

// Notification represents a notification
type Notification struct {
	ID        string           `json:"id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Link      string           `json:"link,omitempty"`
	Read      bool             `json:"read"`
	CreatedAt time.Time        `json:"created_at"`
}

// NotificationService manages notifications for users and admins
type NotificationService struct {
	// In-memory storage for active sessions
	// Persistent storage is handled by the database
	toasts    map[string][]*Notification // userID -> pending toasts
	banners   map[string][]*Notification // userID -> active banners
	toastsMu  sync.RWMutex
	bannersMu sync.RWMutex

	// Configuration
	Position        string        // top-right, top-left, bottom-right, bottom-left
	DefaultDuration time.Duration // default auto-dismiss time
	MaxNotifications int          // max notifications per user
	RetentionDays    int          // days to keep in notification center
}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{
		toasts:           make(map[string][]*Notification),
		banners:          make(map[string][]*Notification),
		Position:         "top-right",
		DefaultDuration:  5 * time.Second,
		MaxNotifications: 100,
		RetentionDays:    30,
	}
}

// AddToast adds a toast notification for a user
func (s *NotificationService) AddToast(userID string, notif *Notification) {
	s.toastsMu.Lock()
	defer s.toastsMu.Unlock()

	if notif.ID == "" {
		notif.ID = generateNotificationID()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}

	s.toasts[userID] = append(s.toasts[userID], notif)
}

// GetToasts returns and clears pending toasts for a user
func (s *NotificationService) GetToasts(userID string) []*Notification {
	s.toastsMu.Lock()
	defer s.toastsMu.Unlock()

	toasts := s.toasts[userID]
	delete(s.toasts, userID)
	return toasts
}

// AddBanner adds a persistent banner for a user
func (s *NotificationService) AddBanner(userID string, notif *Notification) {
	s.bannersMu.Lock()
	defer s.bannersMu.Unlock()

	if notif.ID == "" {
		notif.ID = generateNotificationID()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}

	// Check if banner with same title already exists
	for _, existing := range s.banners[userID] {
		if existing.Title == notif.Title {
			// Update existing banner
			existing.Message = notif.Message
			existing.Link = notif.Link
			return
		}
	}

	s.banners[userID] = append(s.banners[userID], notif)
}

// GetBanners returns active banners for a user
func (s *NotificationService) GetBanners(userID string) []*Notification {
	s.bannersMu.RLock()
	defer s.bannersMu.RUnlock()
	return s.banners[userID]
}

// DismissBanner removes a banner for a user
func (s *NotificationService) DismissBanner(userID, bannerID string) {
	s.bannersMu.Lock()
	defer s.bannersMu.Unlock()

	banners := s.banners[userID]
	for i, b := range banners {
		if b.ID == bannerID {
			s.banners[userID] = append(banners[:i], banners[i+1:]...)
			return
		}
	}
}

// ClearBanners removes all banners for a user
func (s *NotificationService) ClearBanners(userID string) {
	s.bannersMu.Lock()
	defer s.bannersMu.Unlock()
	delete(s.banners, userID)
}

// NotifySuccess sends a success notification
func (s *NotificationService) NotifySuccess(userID, title, message string) {
	s.AddToast(userID, &Notification{
		Type:    NotifySuccess,
		Title:   title,
		Message: message,
	})
}

// NotifyInfo sends an info notification
func (s *NotificationService) NotifyInfo(userID, title, message string) {
	s.AddToast(userID, &Notification{
		Type:    NotifyInfo,
		Title:   title,
		Message: message,
	})
}

// NotifyWarning sends a warning notification
func (s *NotificationService) NotifyWarning(userID, title, message string) {
	s.AddToast(userID, &Notification{
		Type:    NotifyWarning,
		Title:   title,
		Message: message,
	})
}

// NotifyError sends an error notification
func (s *NotificationService) NotifyError(userID, title, message string) {
	s.AddToast(userID, &Notification{
		Type:    NotifyError,
		Title:   title,
		Message: message,
	})
}

// NotifySecurity sends a security notification
func (s *NotificationService) NotifySecurity(userID, title, message string) {
	s.AddToast(userID, &Notification{
		Type:    NotifySecurity,
		Title:   title,
		Message: message,
	})
}

// ShowBanner shows a persistent banner
func (s *NotificationService) ShowBanner(userID string, notifType NotificationType, title, message, link string) {
	s.AddBanner(userID, &Notification{
		Type:    notifType,
		Title:   title,
		Message: message,
		Link:    link,
	})
}

// ShowSMTPWarning shows the SMTP not configured warning banner
func (s *NotificationService) ShowSMTPWarning(userID, adminPath string) {
	s.ShowBanner(userID, NotifyWarning,
		"SMTP not configured",
		"Email features disabled. Configure SMTP to enable email notifications.",
		"/"+adminPath+"/server/email",
	)
}

// ShowSSLExpiringWarning shows the SSL expiring warning banner
func (s *NotificationService) ShowSSLExpiringWarning(userID, adminPath string, daysLeft int) {
	s.ShowBanner(userID, NotifyWarning,
		"SSL Certificate Expiring",
		formatDaysLeft(daysLeft),
		"/"+adminPath+"/server/ssl",
	)
}

// ShowDiskSpaceWarning shows the disk space warning banner
func (s *NotificationService) ShowDiskSpaceWarning(userID, adminPath string, percentFree int) {
	s.ShowBanner(userID, NotifyWarning,
		"Low Disk Space",
		formatDiskSpace(percentFree),
		"/"+adminPath+"/server/info",
	)
}

// generateNotificationID generates a unique notification ID
func generateNotificationID() string {
	return "notif_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random alphanumeric string
func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(b)
}

// formatDaysLeft formats days left message
func formatDaysLeft(days int) string {
	if days == 1 {
		return "Certificate expires in 1 day"
	}
	return "Certificate expires in " + string(rune('0'+days%10)) + " days"
}

// formatDiskSpace formats disk space message
func formatDiskSpace(percentFree int) string {
	return "Only " + string(rune('0'+percentFree%10)) + "% disk space remaining"
}
