package quiz

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(70)

	headerStyle = lipgloss.NewStyle().Bold(true)
	gradeStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

// QuizItem is a single question in the session.
type QuizItem struct {
	NotePath     string
	NoteTitle    string
	Domain       string
	TopicCluster string
	Question     string
	NoteBody     string // for grading context
}

// GradeResult is the grading outcome.
type GradeResult struct {
	Score         int
	CorrectPoints []string
	MissingPoints []string
	Feedback      string
	SM2Grade      int
	NewInterval   int
	NewState      string
}

type state int

const (
	stateQuestion state = iota
	stateTyping
	stateGrading
	stateResult
	stateDone
)

// Model is the Bubbletea model for the quiz TUI.
type Model struct {
	items       []QuizItem
	current     int
	answer      string
	state       state
	grade       *GradeResult
	totalItems  int
	domainStats map[string][]int // domain → list of scores

	// Callbacks set by the CLI.
	OnGrade func(item QuizItem, answer string) (*GradeResult, error)
	OnSkip  func(item QuizItem)
	OnDefer func(item QuizItem)
}

// NewModel creates a quiz TUI model.
func NewModel(items []QuizItem) Model {
	stats := make(map[string][]int)
	return Model{
		items:       items,
		totalItems:  len(items),
		state:       stateQuestion,
		domainStats: stats,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateQuestion, stateTyping:
			return m.handleTyping(msg)
		case stateResult:
			return m.handleResult(msg)
		case stateDone:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.state = stateDone
		return m, tea.Quit
	case "enter":
		if m.state == stateQuestion {
			m.state = stateTyping
			return m, nil
		}
		// Double enter submits.
		if strings.HasSuffix(m.answer, "\n") {
			return m.submitAnswer()
		}
		m.answer += "\n"
	case "backspace":
		if len(m.answer) > 0 {
			m.answer = m.answer[:len(m.answer)-1]
		}
	default:
		key := msg.String()
		if key == ":skip" || m.answer == ":skip" {
			m.answer = ""
			if m.OnSkip != nil {
				m.OnSkip(m.items[m.current])
			}
			return m.nextQuestion()
		}
		if key == ":defer" || m.answer == ":defer" {
			m.answer = ""
			if m.OnDefer != nil {
				m.OnDefer(m.items[m.current])
			}
			return m.nextQuestion()
		}
		if key == ":quit" || m.answer == ":quit" {
			m.state = stateDone
			return m, tea.Quit
		}
		if len(key) == 1 {
			m.state = stateTyping
			m.answer += key
		}
	}
	return m, nil
}

func (m Model) submitAnswer() (tea.Model, tea.Cmd) {
	answer := strings.TrimSpace(m.answer)
	if answer == "" {
		return m, nil
	}

	m.state = stateGrading

	if m.OnGrade != nil {
		grade, err := m.OnGrade(m.items[m.current], answer)
		if err != nil {
			// Self-assessment fallback.
			m.grade = &GradeResult{Score: 50, Feedback: "Grading unavailable. Self-assess.", SM2Grade: 3}
		} else {
			m.grade = grade
		}
	}

	item := m.items[m.current]
	if m.grade != nil {
		m.domainStats[item.Domain] = append(m.domainStats[item.Domain], m.grade.Score)
	}

	m.state = stateResult
	return m, nil
}

func (m Model) handleResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.state = stateDone
		return m, tea.Quit
	case "enter", " ":
		return m.nextQuestion()
	}
	return m, nil
}

func (m Model) nextQuestion() (tea.Model, tea.Cmd) {
	m.current++
	m.answer = ""
	m.grade = nil
	if m.current >= len(m.items) {
		m.state = stateDone
		return m, tea.Quit
	}
	m.state = stateQuestion
	return m, nil
}

func (m Model) View() tea.View {
	switch m.state {
	case stateQuestion, stateTyping:
		return tea.NewView(m.viewQuestion())
	case stateGrading:
		return tea.NewView("\n  Grading...\n")
	case stateResult:
		return tea.NewView(m.viewResult())
	case stateDone:
		return tea.NewView(m.viewSummary())
	}
	return tea.NewView("")
}

func (m Model) viewQuestion() string {
	if m.current >= len(m.items) {
		return ""
	}
	item := m.items[m.current]

	header := fmt.Sprintf("Q%d/%d  %s  %s",
		m.current+1, m.totalItems,
		item.Domain, item.TopicCluster)

	body := fmt.Sprintf("%s\n\n%s", headerStyle.Render(header), item.Question)
	box := boxStyle.Render(body)

	prompt := dimStyle.Render("Your answer (Enter twice to submit, :skip, :defer, :quit):")
	cursor := fmt.Sprintf("\n%s\n> %s_", prompt, m.answer)

	return box + cursor
}

func (m Model) viewResult() string {
	if m.grade == nil {
		return ""
	}

	letter := gradeLetter(m.grade.Score)
	header := fmt.Sprintf("GRADE: %s (%d/100)  SM-2: %d", letter, m.grade.Score, m.grade.SM2Grade)

	var lines []string
	lines = append(lines, gradeStyle.Render(header))

	for _, p := range m.grade.CorrectPoints {
		lines = append(lines, fmt.Sprintf("  ✓ %s", p))
	}
	for _, p := range m.grade.MissingPoints {
		lines = append(lines, fmt.Sprintf("  ✗ %s", p))
	}
	if m.grade.Feedback != "" {
		lines = append(lines, "")
		lines = append(lines, m.grade.Feedback)
	}
	if m.grade.NewState != "" {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Retention: %s, interval %d days", m.grade.NewState, m.grade.NewInterval)))
	}

	box := boxStyle.Render(strings.Join(lines, "\n"))
	return box + "\n" + dimStyle.Render("Press Enter for next question")
}

func (m Model) viewSummary() string {
	if len(m.domainStats) == 0 {
		return "\nNo questions answered.\n"
	}

	var lines []string
	lines = append(lines, headerStyle.Render("SESSION COMPLETE"))
	lines = append(lines, "")

	total := 0
	totalScore := 0
	for domain, scores := range m.domainStats {
		avg := 0
		for _, s := range scores {
			avg += s
			totalScore += s
		}
		avg /= len(scores)
		total += len(scores)
		lines = append(lines, fmt.Sprintf("  %s: %d%% (%d questions)", domain, avg, len(scores)))
	}
	if total > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Overall: %d%% (%d questions)", totalScore/total, total))
	}

	return boxStyle.Render(strings.Join(lines, "\n")) + "\n"
}

func gradeLetter(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "A-"
	case score >= 80:
		return "B+"
	case score >= 75:
		return "B"
	case score >= 70:
		return "B-"
	case score >= 65:
		return "C+"
	case score >= 60:
		return "C"
	case score >= 50:
		return "D"
	default:
		return "F"
	}
}
