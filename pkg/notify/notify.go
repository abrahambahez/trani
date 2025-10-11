package notify

import (
	"fmt"
	"os/exec"
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

// Error sends an error desktop notification with critical urgency.
func (n *Notifier) Error(title, message string) error {
	cmd := exec.Command("notify-send", "-u", "critical", title, message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send error notification: %w", err)
	}
	return nil
}
