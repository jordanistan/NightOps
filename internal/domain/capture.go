package domain

import "strings"

// CaptureProfile is a reusable starting point for imaging a target class.
// These are recommendations, not camera-specific measurements; the operator
// should adjust them for sensor response, sky brightness, and tracking.
type CaptureProfile struct {
	Guidance string
	Settings string
}

// CaptureProfileForKind returns the repeatable starting settings used by both
// the active-mission TUI and the generated Obsidian target notes.
func CaptureProfileForKind(kind string) CaptureProfile {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "galaxy":
		return CaptureProfile{
			Guidance: "Use a wider field and moderate integration; preserve the bright core while exposing faint outer structure.",
			Settings: "Broadband / UV-IR filter; 120–180 s subs; low or unity gain; 30–60 subs; 1.5–3 h total; dither every 3–5 frames.",
		}
	case "nebula":
		return CaptureProfile{
			Guidance: "Use a narrowband or light-pollution-aware filter when available; capture multiple shorter subs for the brightest regions.",
			Settings: "Dual-band / narrowband filter when available; 30–120 s subs plus 5–15 s HDR subs; unity gain; 60–120 subs; 1–3 h total.",
		}
	case "cluster", "globular cluster", "open cluster":
		return CaptureProfile{
			Guidance: "Use a field of view that includes the surrounding star field; keep stars sharp with short, well-focused subs.",
			Settings: "Broadband filter; 30–90 s subs; low gain / ISO; 30–60 subs; 30–90 min total; prioritize round, unsaturated stars.",
		}
	default:
		return CaptureProfile{
			Guidance: "Confirm focus, framing, and exposure with a short test capture before committing the sequence.",
			Settings: "Start with 60 s subs at low or unity gain; capture 10 test subs, inspect stars and histogram, then extend the sequence.",
		}
	}
}
