package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalidLocation is returned when an invalid location value is provided
var ErrInvalidLocation = errors.New("invalid location value")

// Location represents an immutable warehouse location value object
// Format: ZONE-AISLE-RACK-LEVEL (e.g., "A-01-R05-L02")
type Location struct {
	locationID string
	zone       string
	aisle      string
	rack       int
	level      int
}

// locationPattern validates location format: ZONE-AISLE-RACK-LEVEL
var locationPattern = regexp.MustCompile(`^([A-Z]{1,2})-(\d{2})-R(\d{2})-L(\d{2})$`)

// NewLocation creates a new Location value object with validation
func NewLocation(locationID string) (Location, error) {
	if locationID == "" {
		return Location{}, ErrInvalidLocation
	}

	// Parse location format
	matches := locationPattern.FindStringSubmatch(locationID)
	if matches == nil {
		return Location{}, fmt.Errorf("%w: invalid format, expected ZONE-AISLE-RACK-LEVEL", ErrInvalidLocation)
	}

	// Extract components
	zone := matches[1]
	aisle := matches[2]
	rack, _ := strconv.Atoi(matches[3])
	level, _ := strconv.Atoi(matches[4])

	// Validate ranges
	if rack < 1 || rack > 99 {
		return Location{}, fmt.Errorf("%w: rack must be between 01 and 99", ErrInvalidLocation)
	}
	if level < 1 || level > 99 {
		return Location{}, fmt.Errorf("%w: level must be between 01 and 99", ErrInvalidLocation)
	}

	return Location{
		locationID: locationID,
		zone:       zone,
		aisle:      aisle,
		rack:       rack,
		level:      level,
	}, nil
}

// NewLocationFromComponents creates a Location from individual components
func NewLocationFromComponents(zone, aisle string, rack, level int) (Location, error) {
	// Validate zone (1-2 uppercase letters)
	if !regexp.MustCompile(`^[A-Z]{1,2}$`).MatchString(zone) {
		return Location{}, fmt.Errorf("%w: zone must be 1-2 uppercase letters", ErrInvalidLocation)
	}

	// Validate aisle (2 digits)
	if !regexp.MustCompile(`^\d{2}$`).MatchString(aisle) {
		return Location{}, fmt.Errorf("%w: aisle must be 2 digits", ErrInvalidLocation)
	}

	// Validate ranges
	if rack < 1 || rack > 99 {
		return Location{}, fmt.Errorf("%w: rack must be between 1 and 99", ErrInvalidLocation)
	}
	if level < 1 || level > 99 {
		return Location{}, fmt.Errorf("%w: level must be between 1 and 99", ErrInvalidLocation)
	}

	locationID := fmt.Sprintf("%s-%s-R%02d-L%02d", zone, aisle, rack, level)
	return Location{
		locationID: locationID,
		zone:       zone,
		aisle:      aisle,
		rack:       rack,
		level:      level,
	}, nil
}

// MustNewLocation creates a Location or panics if invalid (use for constants only)
func MustNewLocation(locationID string) Location {
	location, err := NewLocation(locationID)
	if err != nil {
		panic(err)
	}
	return location
}

// LocationID returns the full location identifier
func (l Location) LocationID() string {
	return l.locationID
}

// Zone returns the zone component
func (l Location) Zone() string {
	return l.zone
}

// Aisle returns the aisle component
func (l Location) Aisle() string {
	return l.aisle
}

// Rack returns the rack number
func (l Location) Rack() int {
	return l.rack
}

// Level returns the level number
func (l Location) Level() int {
	return l.level
}

// String returns the string representation of the location
func (l Location) String() string {
	return l.locationID
}

// Equals checks if two locations are equal
func (l Location) Equals(other Location) bool {
	return l.locationID == other.locationID
}

// IsSameZone checks if this location is in the same zone as another
func (l Location) IsSameZone(other Location) bool {
	return l.zone == other.zone
}

// IsSameAisle checks if this location is in the same aisle as another
func (l Location) IsSameAisle(other Location) bool {
	return l.zone == other.zone && l.aisle == other.aisle
}

// IsAbove checks if this location is above another (same aisle, higher level)
func (l Location) IsAbove(other Location) bool {
	return l.IsSameAisle(other) && l.rack == other.rack && l.level > other.level
}

// IsBelow checks if this location is below another (same aisle, lower level)
func (l Location) IsBelow(other Location) bool {
	return l.IsSameAisle(other) && l.rack == other.rack && l.level < other.level
}

// IsGroundLevel returns true if this location is on ground level (level 01)
func (l Location) IsGroundLevel() bool {
	return l.level == 1
}

// IsHighLevel returns true if this location is on a high level (level > 5)
func (l Location) IsHighLevel() bool {
	return l.level > 5
}

// DistanceFrom calculates a simple distance metric to another location
// Returns Manhattan distance: |zone_diff| + |aisle_diff| + |rack_diff| + |level_diff|
func (l Location) DistanceFrom(other Location) int {
	zoneDiff := 0
	if l.zone != other.zone {
		// Different zones are far apart
		zoneDiff = 100
	}

	aisleDiff := 0
	if l.aisle != other.aisle {
		// Parse aisle numbers for distance calculation
		aisle1, _ := strconv.Atoi(l.aisle)
		aisle2, _ := strconv.Atoi(other.aisle)
		aisleDiff = abs(aisle1 - aisle2)
	}

	rackDiff := abs(l.rack - other.rack)
	levelDiff := abs(l.level - other.level)

	return zoneDiff + aisleDiff + rackDiff + levelDiff
}

// MarshalText implements encoding.TextMarshaler for JSON/BSON serialization
func (l Location) MarshalText() ([]byte, error) {
	return []byte(l.locationID), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/BSON deserialization
func (l *Location) UnmarshalText(text []byte) error {
	location, err := NewLocation(string(text))
	if err != nil {
		return err
	}
	*l = location
	return nil
}

// ParseLocation is an alias for NewLocation for clarity
func ParseLocation(locationID string) (Location, error) {
	return NewLocation(locationID)
}

// ParseLocationOrSimple attempts to parse a full location, or creates a simple one
// Simple format: just the locationID string without validation (for backward compatibility)
func ParseLocationOrSimple(locationID string) Location {
	loc, err := NewLocation(locationID)
	if err != nil {
		// For backward compatibility, create a simple location with just the ID
		// This allows non-standard location IDs to still work
		return Location{
			locationID: locationID,
			zone:       extractZonePrefix(locationID),
		}
	}
	return loc
}

// extractZonePrefix extracts the first uppercase letter(s) as zone
func extractZonePrefix(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) > 0 {
		return strings.ToUpper(string(parts[0][0]))
	}
	return ""
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
