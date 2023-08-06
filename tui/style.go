package tui

import "github.com/charmbracelet/lipgloss"

const (
	hotPink      = lipgloss.Color("#FF06B7")
	darkGray     = lipgloss.Color("#767676")
	skyBlue      = lipgloss.Color("#4CC9F0")
	deepPink     = lipgloss.Color("#B5179E")
	ceruleanBlue = lipgloss.Color("#05A7E0")
	deepPurple   = lipgloss.Color("#560BAD")

	brighRed = lipgloss.Color("#D00000")
	white    = lipgloss.Color("#000000")
)

var (
	HeaderStyle = lipgloss.NewStyle().Foreground(skyBlue)
	FooterStyle = lipgloss.NewStyle().Foreground(skyBlue)

	tabStyle  = lipgloss.NewStyle().Padding(0, 2)
	TabStyles = []lipgloss.Style{
		tabStyle.Copy().Background(deepPink),
		tabStyle.Copy().Background(ceruleanBlue),
		tabStyle.Background(deepPurple),
	}

	NNInputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	NNContinueStyle = lipgloss.NewStyle().Foreground(darkGray)

	ErrorTextStyle = lipgloss.NewStyle().Background(brighRed).Foreground(white)
)
