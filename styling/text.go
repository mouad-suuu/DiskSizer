package styling

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TextAction represents a clickable action on text
type TextAction struct {
	Text     string
	Callback func()
}

// StyleOptions configures how text should be displayed
type StyleOptions struct {
	Bold            bool
	Italic          bool
	Underline       bool
	Blink           bool
	TextColor       tcell.Color
	BackgroundColor tcell.Color
}

// DefaultStyleOptions returns standard styling
func DefaultStyleOptions() StyleOptions {
	return StyleOptions{
		Bold:            false,
		Italic:          false,
		Underline:       false,
		Blink:           false,
		TextColor:       tcell.ColorWhite,
		BackgroundColor: tcell.ColorDefault,
	}
}

// StyleBuilder facilitates chaining style operations
type StyleBuilder struct {
	options StyleOptions
}

// NewStyleBuilder creates a new style builder with default options
func NewStyleBuilder() *StyleBuilder {
	return &StyleBuilder{
		options: DefaultStyleOptions(),
	}
}

// WithBold sets the bold attribute
func (b *StyleBuilder) WithBold() *StyleBuilder {
	b.options.Bold = true
	return b
}
func getUsageColor(percent float64) tcell.Color {
	// From green (0,255,0) to red (255,0,0)
	red := int(255 * (percent / 100))
	green := int(255 * ((100 - percent) / 100))
	return tcell.NewRGBColor(int32(red), int32(green), 0)
}

// WithItalic sets the italic attribute
func (b *StyleBuilder) WithItalic() *StyleBuilder {
	b.options.Italic = true
	return b
}

// WithUnderline sets the underline attribute
func (b *StyleBuilder) WithUnderline() *StyleBuilder {
	b.options.Underline = true
	return b
}

// WithTextColor sets the text color
func (b *StyleBuilder) WithTextColor(color tcell.Color) *StyleBuilder {
	b.options.TextColor = color
	return b
}

// WithBackgroundColor sets the background color
func (b *StyleBuilder) WithBackgroundColor(color tcell.Color) *StyleBuilder {
	b.options.BackgroundColor = color
	return b
}

// Build creates the final StyleOptions object
func (b *StyleBuilder) Build() StyleOptions {
	return b.options
}

// ActionRegistry manages clickable text actions
type ActionRegistry struct {
	actions map[string]func()
	nextID  int
	mu      sync.Mutex
}

// NewActionRegistry creates a new action registry
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[string]func()),
		nextID:  1,
	}
}

// Register adds a new action and returns its ID
func (r *ActionRegistry) Register(callback func()) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("action-%d", r.nextID)
	r.nextID++
	r.actions[id] = callback
	return id
}

// Execute runs an action by ID if it exists
func (r *ActionRegistry) Execute(id string) bool {
	r.mu.Lock()
	action, exists := r.actions[id]
	r.mu.Unlock()

	if exists {
		action()
		return true
	}
	return false
}

// Global registry instance
var GlobalRegistry = NewActionRegistry()

// ApplyStyle applies styling to text using tview's color tags
func ApplyStyle(text string, style StyleOptions) string {
	// Start with opening color tag
	result := "["

	// Add text color
	if style.TextColor != tcell.ColorDefault {
		result += fmt.Sprintf("#%06x", style.TextColor.Hex())
	}

	// Add background color if specified
	if style.BackgroundColor != tcell.ColorDefault {
		result += ":" + fmt.Sprintf("#%06x", style.BackgroundColor.Hex())
	}

	// Add style attributes
	if style.Bold {
		result += ":b"
	}
	if style.Italic {
		result += ":i"
	}
	if style.Underline {
		result += ":u"
	}
	if style.Blink {
		result += ":l" // 'l' for blink in tview
	}

	// Close the opening tag and add the text
	result += "]" + text

	// Add closing tag
	result += "[-]"

	return result
}

// MakeClickable creates clickable text that executes a callback when selected
func MakeClickable(text string, style StyleOptions, callback func()) string {
	actionID := GlobalRegistry.Register(callback)
	styledText := ApplyStyle(text, style)
	return fmt.Sprintf(`["%s"]%s[""]`, actionID, styledText)
}

// InstallClickHandler installs the handler for clickable text in a TextView
func InstallClickHandler(textView *tview.TextView, app *tview.Application) {
	textView.SetRegions(true) // Enable region support

	// Highlight region when mouse clicks
	textView.SetDoneFunc(func(key tcell.Key) {
		regions := textView.GetHighlights()
		if len(regions) > 0 {
			regionID := regions[0]
			if GlobalRegistry.Execute(regionID) {
				app.Draw()
			}
		}
	})
}

// CreateInfoText creates styled informational text
func CreateInfoText(label, value string, valueColor tcell.Color) string {
	labelStyle := NewStyleBuilder().
		WithBold().
		WithTextColor(tcell.ColorWhite).
		Build()

	valueStyle := NewStyleBuilder().
		WithTextColor(valueColor).
		Build()

	styledLabel := ApplyStyle(label, labelStyle)
	styledValue := ApplyStyle(value, valueStyle)

	return styledLabel + ": " + styledValue
}

// FormatSizeWithColor formats a size value with appropriate color based on usage
func FormatSizeWithColor(size float64, total float64, unit string) string {
	percentUsed := (size / total) * 100

	var color tcell.Color
	if percentUsed > 90 {
		color = tcell.ColorRed
	} else if percentUsed > 70 {
		color = tcell.ColorYellow
	} else {
		color = tcell.ColorGreen
	}

	style := NewStyleBuilder().
		WithTextColor(color).
		Build()

	return ApplyStyle(fmt.Sprintf("%.2f %s", size, unit), style)
}

// CreateProgressBar generates a text-based progress bar
func CreateProgressBar(used float64, total float64, width int) string {
	if total <= 0 {
		return ""
	}

	percentage := used / total
	filledWidth := int(float64(width) * percentage)

	if filledWidth > width {
		filledWidth = width
	}

	// Choose color based on usage percentage
	var barColor tcell.Color
	if percentage > 0.9 {
		barColor = tcell.ColorRed
	} else if percentage > 0.7 {
		barColor = tcell.ColorYellow
	} else {
		barColor = tcell.ColorGreen
	}

	// Create the filled part
	filledStyle := NewStyleBuilder().
		WithTextColor(barColor).
		WithBackgroundColor(barColor).
		Build()
		// Add percentage text
	percentText := fmt.Sprintf(" %.1f%%", percentage*100)
	filled := strings.Repeat("█", filledWidth)
	styledFilled := ApplyStyle((percentText + filled), filledStyle)

	// Create the empty part
	emptyStyle := NewStyleBuilder().
		WithTextColor(tcell.ColorWhite).
		Build()
	empty := strings.Repeat("░", width-filledWidth)
	styledEmpty := ApplyStyle(empty, emptyStyle)

	return styledFilled + styledEmpty
}

// SplitIntoPages splits long text into pages with given height
func SplitIntoPages(text string, linesPerPage int) []string {
	lines := strings.Split(text, "\n")
	var pages []string

	for i := 0; i < len(lines); i += linesPerPage {
		end := i + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}

		pageLines := lines[i:end]
		pages = append(pages, strings.Join(pageLines, "\n"))
	}

	return pages
}

// CreateHeader creates a styled section header
func CreateHeader(text string) string {
	headerStyle := NewStyleBuilder().
		WithBold().
		WithTextColor(tcell.ColorNames["cyan"]).
		Build()

	styledText := ApplyStyle(text, headerStyle)
	line := ApplyStyle(strings.Repeat("─", len(text)+4), headerStyle)

	return "\n" + styledText + "\n" + line
}

// WrapWithAction wraps text with a callback when clicked
func WrapWithAction(textView *tview.TextView, text string, callback func()) string {
	// Make sure textView has mouse capture set up
	if textView != nil {
		if textView.GetMouseCapture() == nil {
			InstallClickHandler(textView, tview.NewApplication())
		}
	}

	// Create a clickable style
	clickableStyle := NewStyleBuilder().
		WithTextColor(tcell.ColorBlue).
		WithUnderline().
		Build()

	return MakeClickable(text, clickableStyle, callback)
}

// CreateListItem formats a list item with optional click action
func CreateListItem(textView *tview.TextView, prefix, text string, callback func()) string {
	if callback != nil {
		return prefix + " " + WrapWithAction(textView, text, callback)
	}

	itemStyle := NewStyleBuilder().
		WithTextColor(tcell.ColorWhite).
		Build()

	return prefix + " " + ApplyStyle(text, itemStyle)
}
