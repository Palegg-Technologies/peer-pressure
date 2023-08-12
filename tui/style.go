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
	HeaderStyle = lipgloss.NewStyle().Foreground(skyBlue).Render
	FooterStyle = lipgloss.NewStyle().Foreground(skyBlue).Render

	tabStyle  = lipgloss.NewStyle().Padding(0, 2)
	TabStyles = []func(strs ...string) string{
		tabStyle.Copy().Background(deepPink).Render,
		tabStyle.Copy().Background(ceruleanBlue).Render,
		tabStyle.Background(deepPurple).Render,
	}

	width           = 30
	NNInputStyle    = lipgloss.NewStyle().Foreground(hotPink).Width(width).Render
	NNContinueStyle = lipgloss.NewStyle().Foreground(darkGray).Width(width).Render

	ErrorTextStyle = lipgloss.NewStyle().Background(brighRed).Foreground(white).Render
)
