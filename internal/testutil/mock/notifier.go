package mock

import (
	"context"

	"github.com/beacon-stack/prism/pkg/plugin"
)

// Notifier is a configurable mock of plugin.Notifier.
type Notifier struct {
	NotifyFunc func(ctx context.Context, event plugin.NotificationEvent) error
	TestFunc   func(ctx context.Context) error

	Calls  []string
	Events []plugin.NotificationEvent
}

func (m *Notifier) Name() string { return "MockNotifier" }

func (m *Notifier) Notify(ctx context.Context, event plugin.NotificationEvent) error {
	m.Calls = append(m.Calls, "Notify")
	m.Events = append(m.Events, event)
	if m.NotifyFunc != nil {
		return m.NotifyFunc(ctx, event)
	}
	return nil
}

func (m *Notifier) Test(ctx context.Context) error {
	m.Calls = append(m.Calls, "Test")
	if m.TestFunc != nil {
		return m.TestFunc(ctx)
	}
	return nil
}
