package domain

import (
	"errors"
	"strings"
	"time"
)

// EquipmentProfile is a reusable observing setup owned by the local user.
type EquipmentProfile struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewEquipmentProfile creates a validated local equipment profile.
func NewEquipmentProfile(id, name, description string, now time.Time) (EquipmentProfile, error) {
	if id == "" || strings.TrimSpace(name) == "" {
		return EquipmentProfile{}, errors.New("equipment profile id and name are required")
	}
	return EquipmentProfile{ID: id, Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), CreatedAt: now, UpdatedAt: now}, nil
}

// EquipmentItem is an inventory item assigned to a reusable equipment profile.
type EquipmentItem struct {
	ID        string
	ProfileID string
	Name      string
	Category  string
	Required  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewEquipmentItem creates a validated inventory item.
func NewEquipmentItem(id, profileID, name, category string, required bool, now time.Time) (EquipmentItem, error) {
	if id == "" || profileID == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(category) == "" {
		return EquipmentItem{}, errors.New("equipment item id, profile, name, and category are required")
	}
	return EquipmentItem{ID: id, ProfileID: profileID, Name: strings.TrimSpace(name), Category: strings.TrimSpace(category), Required: required, CreatedAt: now, UpdatedAt: now}, nil
}
