// Package service — Tests for NotificationService (in-memory toast and banner management).
// Covers: NewNotificationService, NotificationType.Icon, AutoDismiss,
// AddToast, GetToasts (clears after read), AddBanner (create and update existing),
// GetBanners, DismissBanner, ClearBanners,
// NotifySuccess/Info/Warning/Error/Security convenience methods,
// ShowBanner, ShowSMTPWarning, ShowSSLExpiringWarning, ShowDiskSpaceWarning.
package service

import (
	"testing"
	"time"
)

// --- NotificationType.Icon ---

func TestNotificationTypeIcon(t *testing.T) {
	t.Parallel()
	cases := []struct {
		t    NotificationType
		want string
	}{
		{NotifySuccess, "check-circle"},
		{NotifyInfo, "info-circle"},
		{NotifyWarning, "alert-triangle"},
		{NotifyError, "x-circle"},
		{NotifySecurity, "lock"},
		// Unknown type returns "bell"
		{NotificationType("unknown"), "bell"},
	}
	for _, tc := range cases {
		got := tc.t.Icon()
		if got != tc.want {
			t.Errorf("NotificationType(%q).Icon() = %q, want %q", tc.t, got, tc.want)
		}
	}
}

// --- NotificationType.AutoDismiss ---

func TestNotificationTypeAutoDismiss(t *testing.T) {
	t.Parallel()
	cases := []struct {
		t    NotificationType
		want time.Duration
	}{
		{NotifySuccess, 5 * time.Second},
		{NotifyInfo, 5 * time.Second},
		{NotifyWarning, 10 * time.Second},
		// Error and security require manual dismiss
		{NotifyError, 0},
		{NotifySecurity, 0},
		// Unknown defaults to 5s
		{NotificationType("other"), 5 * time.Second},
	}
	for _, tc := range cases {
		got := tc.t.AutoDismiss()
		if got != tc.want {
			t.Errorf("NotificationType(%q).AutoDismiss() = %v, want %v", tc.t, got, tc.want)
		}
	}
}

// --- NewNotificationService ---

func TestNewNotificationServiceReturnsNonNil(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	if svc == nil {
		t.Error("NewNotificationService returned nil")
	}
}

func TestNewNotificationServiceHasSaneDefaults(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	if svc.Position != "top-right" {
		t.Errorf("Position = %q, want top-right", svc.Position)
	}
	if svc.DefaultDuration != 5*time.Second {
		t.Errorf("DefaultDuration = %v, want 5s", svc.DefaultDuration)
	}
	if svc.MaxNotifications != 100 {
		t.Errorf("MaxNotifications = %d, want 100", svc.MaxNotifications)
	}
	if svc.RetentionDays != 30 {
		t.Errorf("RetentionDays = %d, want 30", svc.RetentionDays)
	}
}

// --- AddToast and GetToasts ---

func TestAddToastAndGetToasts(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddToast("user1", &Notification{
		Type:    NotifyInfo,
		Title:   "Test",
		Message: "Hello",
	})
	toasts := svc.GetToasts("user1")
	if len(toasts) != 1 {
		t.Fatalf("GetToasts returned %d toasts, want 1", len(toasts))
	}
	if toasts[0].Title != "Test" {
		t.Errorf("toast.Title = %q, want Test", toasts[0].Title)
	}
}

func TestGetToastsClearsAfterRead(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddToast("user2", &Notification{Type: NotifySuccess, Title: "Done"})

	first := svc.GetToasts("user2")
	if len(first) != 1 {
		t.Fatalf("first GetToasts: %d toasts, want 1", len(first))
	}

	second := svc.GetToasts("user2")
	if len(second) != 0 {
		t.Errorf("second GetToasts: %d toasts, want 0 (consumed)", len(second))
	}
}

func TestAddToastSetsIDIfMissing(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	n := &Notification{Type: NotifyInfo, Title: "No ID"}
	svc.AddToast("u", n)
	if n.ID == "" {
		t.Error("AddToast should set ID when empty")
	}
}

func TestAddToastSetsCreatedAt(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	n := &Notification{Type: NotifyInfo, Title: "No Time"}
	svc.AddToast("u", n)
	if n.CreatedAt.IsZero() {
		t.Error("AddToast should set CreatedAt when zero")
	}
}

func TestGetToastsEmptyUserReturnsNil(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	toasts := svc.GetToasts("nobody")
	if len(toasts) != 0 {
		t.Errorf("GetToasts(unknown user) returned %d, want 0", len(toasts))
	}
}

// --- AddBanner and GetBanners ---

func TestAddBannerAndGetBanners(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddBanner("admin1", &Notification{
		Type:    NotifyWarning,
		Title:   "Alert",
		Message: "Something is wrong",
	})
	banners := svc.GetBanners("admin1")
	if len(banners) != 1 {
		t.Fatalf("GetBanners returned %d, want 1", len(banners))
	}
	if banners[0].Title != "Alert" {
		t.Errorf("banner.Title = %q, want Alert", banners[0].Title)
	}
}

func TestAddBannerUpdatesExistingByTitle(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddBanner("u", &Notification{Title: "SSL Expiring", Message: "in 30 days"})
	svc.AddBanner("u", &Notification{Title: "SSL Expiring", Message: "in 7 days"})

	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Fatalf("duplicate title should update in-place, got %d banners", len(banners))
	}
	if banners[0].Message != "in 7 days" {
		t.Errorf("updated message = %q, want 'in 7 days'", banners[0].Message)
	}
}

func TestGetBannersDoesNotClearBanners(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddBanner("u", &Notification{Title: "Persistent"})

	svc.GetBanners("u")
	remaining := svc.GetBanners("u")
	if len(remaining) != 1 {
		t.Errorf("banners should persist after GetBanners, got %d", len(remaining))
	}
}

// --- DismissBanner ---

func TestDismissBannerRemovesEntry(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	n := &Notification{ID: "banner-1", Title: "Dismiss Me", Type: NotifyInfo}
	svc.AddBanner("u", n)

	svc.DismissBanner("u", "banner-1")
	banners := svc.GetBanners("u")
	if len(banners) != 0 {
		t.Errorf("after DismissBanner: %d banners, want 0", len(banners))
	}
}

func TestDismissBannerUnknownIDIsNoOp(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddBanner("u", &Notification{ID: "real", Title: "Keep me"})

	svc.DismissBanner("u", "nonexistent")
	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Errorf("DismissBanner(unknown id) removed %d banners, want 0 removed", 1-len(banners))
	}
}

// --- ClearBanners ---

func TestClearBannersRemovesAll(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.AddBanner("u", &Notification{Title: "B1"})
	svc.AddBanner("u", &Notification{Title: "B2"})
	svc.ClearBanners("u")
	if banners := svc.GetBanners("u"); len(banners) != 0 {
		t.Errorf("after ClearBanners: %d banners, want 0", len(banners))
	}
}

// --- Convenience notify methods ---

func TestNotifySuccessCreatesToast(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifySuccess("u", "Done", "Operation succeeded")
	toasts := svc.GetToasts("u")
	if len(toasts) != 1 || toasts[0].Type != NotifySuccess {
		t.Errorf("NotifySuccess did not create a success toast: %+v", toasts)
	}
}

func TestNotifyInfoCreatesToast(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifyInfo("u", "Info", "For your information")
	toasts := svc.GetToasts("u")
	if len(toasts) != 1 || toasts[0].Type != NotifyInfo {
		t.Errorf("NotifyInfo did not create an info toast: %+v", toasts)
	}
}

func TestNotifyWarningCreatesToast(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifyWarning("u", "Warn", "Warning message")
	toasts := svc.GetToasts("u")
	if len(toasts) != 1 || toasts[0].Type != NotifyWarning {
		t.Errorf("NotifyWarning did not create a warning toast: %+v", toasts)
	}
}

func TestNotifyErrorCreatesToast(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifyError("u", "Error", "Something failed")
	toasts := svc.GetToasts("u")
	if len(toasts) != 1 || toasts[0].Type != NotifyError {
		t.Errorf("NotifyError did not create an error toast: %+v", toasts)
	}
}

func TestNotifySecurityCreatesToast(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifySecurity("u", "Security Alert", "Suspicious login attempt")
	toasts := svc.GetToasts("u")
	if len(toasts) != 1 || toasts[0].Type != NotifySecurity {
		t.Errorf("NotifySecurity did not create a security toast: %+v", toasts)
	}
}

// --- ShowBanner ---

func TestShowBannerCreatesBanner(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.ShowBanner("u", NotifyWarning, "Low Space", "10% remaining", "/admin/disk")
	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Fatalf("ShowBanner: %d banners, want 1", len(banners))
	}
	if banners[0].Link != "/admin/disk" {
		t.Errorf("banner.Link = %q, want /admin/disk", banners[0].Link)
	}
}

// --- ShowSMTPWarning ---

func TestShowSMTPWarningCreatesBanner(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.ShowSMTPWarning("u", "admin")
	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Fatalf("ShowSMTPWarning: %d banners, want 1", len(banners))
	}
	if banners[0].Title != "SMTP not configured" {
		t.Errorf("SMTP warning title = %q", banners[0].Title)
	}
}

// --- ShowSSLExpiringWarning ---

func TestShowSSLExpiringWarningCreatesBanner(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.ShowSSLExpiringWarning("u", "admin", 7)
	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Fatalf("ShowSSLExpiringWarning: %d banners, want 1", len(banners))
	}
}

// --- ShowDiskSpaceWarning ---

func TestShowDiskSpaceWarningCreatesBanner(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.ShowDiskSpaceWarning("u", "admin", 5)
	banners := svc.GetBanners("u")
	if len(banners) != 1 {
		t.Fatalf("ShowDiskSpaceWarning: %d banners, want 1", len(banners))
	}
}

// --- Multiple users are independent ---

func TestToastsArePerUserIsolated(t *testing.T) {
	t.Parallel()
	svc := NewNotificationService()
	svc.NotifyInfo("user-a", "A title", "A message")
	svc.NotifyInfo("user-b", "B title", "B message")

	aToasts := svc.GetToasts("user-a")
	bToasts := svc.GetToasts("user-b")

	if len(aToasts) != 1 || aToasts[0].Title != "A title" {
		t.Errorf("user-a got wrong toasts: %+v", aToasts)
	}
	if len(bToasts) != 1 || bToasts[0].Title != "B title" {
		t.Errorf("user-b got wrong toasts: %+v", bToasts)
	}
}
