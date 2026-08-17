package notify

import (
	"fmt"
	"os/exec"
	"strings"
)

// Notifier sends desktop notifications using notify-send.
type Notifier struct{}

// New creates a new Notifier instance.
func New() *Notifier {
	return &Notifier{}
}

// Info sends an informational desktop notification.
func (n *Notifier) Info(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

// Start sends a notification and returns its ID so it can later be updated
// in place with Update instead of stacking a new notification.
func (n *Notifier) Start(title, message string) (string, error) {
	cmd := exec.Command("notify-send", "-p", title, message)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to send notification: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Update replaces the notification with the given ID, reflecting a new state.
func (n *Notifier) Update(id, title, message string) error {
	cmd := exec.Command("notify-send", "-r", id, title, message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}
	return nil
}

// Error sends an error desktop notification with critical urgency.
func (n *Notifier) Error(title, message string) error {
	cmd := exec.Command("notify-send", "-u", "critical", title, message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send error notification: %w", err)
	}
	return nil
}
