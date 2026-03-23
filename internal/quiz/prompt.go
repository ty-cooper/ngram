package quiz

import (
	"fmt"
	"strings"
)

// BuildQuestionPrompt generates a domain-adaptive question from a note.
func BuildQuestionPrompt(title, domain, topicCluster, body string) string {
	var b strings.Builder

	b.WriteString("You are generating quiz questions to test deep knowledge retention.\n")
	b.WriteString("Test UNDERSTANDING, not recall. The student wrote these notes themselves.\n\n")

	fmt.Fprintf(&b, "Note domain: %s, cluster: %s.\n\n", domain, topicCluster)

	b.WriteString("Adapt your question style:\n")
	b.WriteString("- Practical/applied domains (pentest, devops): scenario-based, real-world decisions\n")
	b.WriteString("- Theoretical domains (distributed-systems, algorithms): \"why\" and \"what happens when\"\n")
	b.WriteString("- Engineering domains (data-engineering, systems-design): tradeoff and design questions\n\n")

	b.WriteString("Rules:\n")
	b.WriteString("- Generate 1 medium difficulty question\n")
	b.WriteString("- Answerable from the note content alone\n")
	b.WriteString("- Never yes/no or multiple-choice\n\n")

	b.WriteString("Return ONLY valid JSON:\n")
	b.WriteString(`{"question": "...", "difficulty": "medium", "concepts_tested": ["c1", "c2"], "ideal_answer_points": ["p1", "p2"]}`)
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "NOTE TITLE: %s\n\nNOTE CONTENT:\n%s", title, body)
	return b.String()
}

// BuildGradingPrompt generates a grading prompt for an answer.
func BuildGradingPrompt(question, answer, noteBody, domain string) string {
	var b strings.Builder

	b.WriteString("You are grading a knowledge recall answer. Be rigorous but fair.\n\n")
	fmt.Fprintf(&b, "Domain: %s\n\n", domain)

	b.WriteString("Grade on:\n")
	b.WriteString("1. ACCURACY: Is the answer technically correct?\n")
	b.WriteString("2. COMPLETENESS: Does it cover the key points?\n")
	b.WriteString("3. DEPTH: Does the student understand WHY, not just WHAT?\n")
	b.WriteString("4. APPLICATION: Could they apply this in practice?\n\n")

	b.WriteString("Score 0-100. A surface-level answer that names the right concept but can't explain the mechanism is a C at best.\n\n")

	b.WriteString("Return ONLY valid JSON:\n")
	b.WriteString(`{"score": 0, "correct_points": ["..."], "missing_points": ["..."], "misconceptions": ["..."], "feedback": "2-3 sentence assessment"}`)
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "QUESTION: %s\n\nSTUDENT ANSWER: %s\n\nSOURCE NOTE:\n%s", question, answer, noteBody)
	return b.String()
}

// FallbackQuestion returns a template question when LLM is unavailable.
func FallbackQuestion(title string) string {
	return fmt.Sprintf("Explain %q in your own words. Cover the key concepts and why they matter.", title)
}
