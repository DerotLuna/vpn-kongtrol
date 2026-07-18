package main

// ═══════════════════════════════════════════════════════════════════════════
// "Signal Contour" CLI theme — v2.0.0 · 2026-07-14
// Same palette as the landing site (landing/src/index.css): amber phosphor is
// the live signal, steel is the cool secondary, everything else is cool ink.
// Truecolor hex; lipgloss/termenv downsample on 256/16-color terminals and
// NO_COLOR / --plain strip styling entirely.
// ═══════════════════════════════════════════════════════════════════════════

import "github.com/charmbracelet/lipgloss"

// ── Palette tokens ────────────────────────────────────────────────────────────

var (
	colSignal   = lipgloss.Color("#ffb020") // live signal — connected / armed / ok
	colSignalHi = lipgloss.Color("#ffc456") // highlight / warnings
	colSteel    = lipgloss.Color("#6ea8c9") // cool secondary — info / connecting / IPs
	colText     = lipgloss.Color("#eef1f5")
	colMuted    = lipgloss.Color("#99a3b3")
	colDim      = lipgloss.Color("#616b7b")
	colDanger   = lipgloss.Color("#ff5a52")
	colRule     = lipgloss.Color("#313847") // separator rules / borders
	colBarBg    = lipgloss.Color("#171b23") // tmux bar background

	// amber embers for the banner ramp and secondary amber accents
	colEmber2 = lipgloss.Color("#a8690f")
	colEmber3 = lipgloss.Color("#d18a16")
)

// ── Semantic styles — the single source for every command ────────────────────

var (
	styleOK     = lipgloss.NewStyle().Foreground(colSignal).Bold(true)
	styleErr    = lipgloss.NewStyle().Foreground(colDanger).Bold(true)
	styleWarn   = lipgloss.NewStyle().Foreground(colSignalHi).Bold(true)
	styleInfo   = lipgloss.NewStyle().Foreground(colSteel)
	stylePrompt = lipgloss.NewStyle().Foreground(colSignal).Bold(true)
	styleDim    = lipgloss.NewStyle().Foreground(colDim)
	styleMuted  = lipgloss.NewStyle().Foreground(colMuted)
	styleBright = lipgloss.NewStyle().Foreground(colText).Bold(true)
	styleGold   = lipgloss.NewStyle().Foreground(colSignalHi).Bold(true)
)

// titleGradient holds per-letter colors: K O N G T R O L (ember → signal → ember)
var titleGradient = []lipgloss.Color{
	"#d18a16", "#eda31c", "#ffb020", "#ffc456",
	"#ffc456", "#ffb020", "#eda31c", "#d18a16",
}

// status / map table styles
var (
	styleStatusHdr  = lipgloss.NewStyle().Foreground(colDim).Bold(true)
	styleStatusName = lipgloss.NewStyle().Foreground(colText).Bold(true)
	styleStatusUp   = lipgloss.NewStyle().Foreground(colSignal).Bold(true)
	styleStatusDown = lipgloss.NewStyle().Foreground(colDim)
	styleStatusErr  = lipgloss.NewStyle().Foreground(colDanger).Bold(true)
	styleStatusIP   = lipgloss.NewStyle().Foreground(colSteel)
	styleStatusTime = lipgloss.NewStyle().Foreground(colMuted)

	styleMapHdr      = lipgloss.NewStyle().Foreground(colDim).Bold(true)
	styleMapName     = lipgloss.NewStyle().Foreground(colText).Bold(true)
	styleMapDomain   = lipgloss.NewStyle().Foreground(colSteel)
	styleMapIP       = lipgloss.NewStyle().Foreground(colMuted)
	styleMapVia      = lipgloss.NewStyle().Foreground(colSignal).Bold(true)
	styleMapResolved = lipgloss.NewStyle().Foreground(colSignal).Bold(true)
)

// legacy color shims (older call sites still reference these)
var (
	cDim    = colDim
	cBright = colText
)
