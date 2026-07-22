// Package application defines use-case boundaries between NightOps domains and adapters.
package application

import (
	"context"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
)

// Clock makes time-dependent use cases deterministic in tests.
type Clock interface{ Now() time.Time }

// MissionRepository persists mission aggregates.
type MissionRepository interface {
	Save(context.Context, domain.Mission) error
	Find(context.Context, string) (domain.Mission, error)
}

// LaunchSiteRepository persists and queries observing locations.
type LaunchSiteRepository interface {
	Save(context.Context, domain.LaunchSite) error
	Find(context.Context, string) (domain.LaunchSite, error)
}

// MissionExporter projects a mission into an external knowledge system.
type MissionExporter interface {
	Export(context.Context, domain.Mission, domain.LaunchSite) error
}
