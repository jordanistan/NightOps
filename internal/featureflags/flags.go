// Package featureflags provides explicit capability checks for incomplete systems.
package featureflags

// Flags is the runtime feature set.
type Flags struct {
	CommandPalette bool
	Atlas          bool
	Weather        bool
	Plugins        bool
}

// Enabled reports whether a named feature is enabled.
func (f Flags) Enabled(name string) bool {
	switch name {
	case "command_palette":
		return f.CommandPalette
	case "atlas":
		return f.Atlas
	case "weather":
		return f.Weather
	case "plugins":
		return f.Plugins
	default:
		return false
	}
}
