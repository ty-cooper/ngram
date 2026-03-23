# Spaced Repetition

Notes accumulate but there's no mechanism to verify retention. This system quizzes you on `content_type: knowledge` notes via SM-2 spaced repetition, delivered through iMessage and the terminal.

## Design Principles

- You don't choose what to study. The algorithm chooses based on what's due, what's weak, and what's new.
- Domain-agnostic. The same engine handles pentest techniques, distributed systems theory, recipes, anything.
- Questions generated fresh each session via Claude (prevents memorizing answers instead of concepts).

## Note Lifecycle

```
NEW → LEARNING → REVIEWING → SOLIDIFIED
         ↑            |
         └────────────┘  (failed recall → back to LEARNING)
```

- **NEW**: Just entered the vault. Never quizzed.
- **LEARNING**: Short intervals. Fresh or recently failed.
- **REVIEWING**: Increasing intervals. Passed initial learning.
- **SOLIDIFIED**: 5+ consecutive correct at 30+ day intervals. Spot-checked quarterly (90-day cap).

## The Algorithm (SM-2)

Grade scale: 0 = blackout, 1 = wrong but recognized topic, 2 = wrong but familiar, 3 = correct with difficulty, 4 = correct with hesitation, 5 = perfect recall.

```go
func UpdateRetention(r *Retention, grade int) {
    r.TotalReviews++

    if grade >= 3 { // PASS
        r.TotalCorrect++
        r.Streak++

        switch r.State {
        case "new":
            r.State = "learning"
            r.IntervalDays = 1
            r.RepetitionCount = 1
        case "learning":
            switch {
            case r.RepetitionCount == 0:
                r.IntervalDays = 1
            case r.RepetitionCount == 1:
                r.IntervalDays = 3
            default:
                r.State = "reviewing"
                r.IntervalDays = 7
            }
            r.RepetitionCount++
        case "reviewing", "solidified":
            r.RepetitionCount++
            r.IntervalDays = int(math.Round(float64(r.IntervalDays) * r.EaseFactor))
            if r.IntervalDays > 90 { r.IntervalDays = 90 }
            if r.State == "reviewing" && r.Streak >= 5 && r.IntervalDays >= 30 {
                r.State = "solidified"
            }
        }

        // SM-2 ease factor update
        r.EaseFactor += 0.1 - float64(5-grade)*(0.08+float64(5-grade)*0.02)
        if r.EaseFactor < 1.3 { r.EaseFactor = 1.3 }

    } else { // FAIL (grade 0-2)
        r.Streak = 0
        r.LapseCount++
        if r.State == "reviewing" || r.State == "solidified" {
            r.State = "learning"
            r.RepetitionCount = 0
        }
        r.IntervalDays = 1
        r.EaseFactor -= 0.2
        if r.EaseFactor < 1.3 { r.EaseFactor = 1.3 }
    }

    r.RetentionScore = int(math.Round(float64(r.TotalCorrect) / float64(r.TotalReviews) * 100))
    r.NextReview = time.Now().AddDate(0, 0, r.IntervalDays).Format("2006-01-02")
    r.LastReviewed = time.Now().UTC().Format(time.RFC3339)
}
```

### Interval Progression

Successful streak (grade 4-5):

```
Day 0:   Created (NEW)
Day 1:   Pass → interval 1   (LEARNING)
Day 2:   Pass → interval 3   (LEARNING)
Day 5:   Pass → interval 7   (REVIEWING)
Day 12:  Pass → interval 18  (REVIEWING, ease 2.5)
Day 30:  Pass → interval 45  (REVIEWING)
Day 75:  Pass → interval 90  (SOLIDIFIED)
Day 165: Pass → interval 90  (SOLIDIFIED, capped)
```

Failure on Day 12:

```
Day 12:  FAIL → interval 1, ease drops (back to LEARNING)
Day 13:  Pass → interval 1   (LEARNING restart)
Day 14:  Pass → interval 3
Day 17:  Pass → interval 7   (back to REVIEWING)
Day 24:  Pass → interval 16  (lower ease = slower growth)
```

## Daily Queue Builder

Priority ordering:

1. **Overdue** (most overdue first)
2. **Due today**
3. **New** (oldest first, max 3 per day)
4. **Lapsed** (lapse_count >= 2, most lapsed first)

Interleaved so no two consecutive questions share the same `domain:topic_cluster`. Forces cold recall across subjects.

## Question Generation

Questions generated fresh per session via Claude. Domain-adaptive framing:

- **Practical domains** (pentest, devops): scenario-based engagement questions
- **Theoretical domains** (distributed-systems, algorithms): "why" and "what happens when" questions
- **Engineering domains** (data-engineering, systems-design): tradeoff and design questions

Three difficulty levels: easy (define), medium (compare/contrast), hard (synthesize/design).

### Grading

Claude grades on accuracy, completeness, depth, and application. Domain-specific weighting (practical domains weight application heavily, theoretical domains weight depth). Score 0-100, translated to SM-2 grade:

```go
func LLMScoreToGrade(llmScore int) int {
    switch {
    case llmScore >= 95: return 5
    case llmScore >= 80: return 4
    case llmScore >= 60: return 3
    case llmScore >= 40: return 2
    case llmScore >= 20: return 1
    default:             return 0
    }
}
```

API degradation: when Claude is unreachable, fall back to self-assessment ("Rate yourself 1-5").

## Terminal Quiz (`n quiz`)

Bubbletea TUI with interactive sessions:

```bash
n quiz                    # system-selected, cross-domain
n quiz --domain pentest   # domain-filtered
n quiz --weak             # retention_score < 60 only
n quiz --new              # unreviewed notes only
```

## iMessage Delivery

Questions arrive as iMessages at random times during waking hours. You text back. Graded in 30 seconds.

### Bridge Architecture

```go
type MessageBridge interface {
    Send(phone string, text string) error
    Poll() ([]IncomingMessage, error)
}
```

Two implementations: AppleScript (MVP, fragile) and BlueBubbles REST API (production, reliable). Swappable via config. Future option: Telegram bot.

### Scheduler

Runs as a goroutine inside `n`. Every 30 seconds:

1. Check if current time matches a scheduled send time → generate question, send
2. Check for timeout on pending questions (4 hours) → grade as 0, send the answer
3. Poll bridge for replies → match to oldest pending (FIFO), grade via Claude, send feedback

State persisted to `_meta/quiz-delivery-state.json` after every change. Survives restarts.

### Special Commands

| Reply | Action |
|-------|--------|
| (answer text) | Grade against source note |
| `skip` | Grade 0 |
| `idk` | Grade 0, system sends the answer |
| `hint` | Get a hint, caps max grade at 3 |
| `pause` | Pause delivery for today |
| `resume` | Resume delivery |
| `stats` | Today's score summary |
| `ask: <question>` | RAG query via iMessage |
| `note: <text>` | Capture a note via iMessage |

### No-Response Handling

4-hour timeout (configurable). No response = grade 0. System sends the answer for passive review.

### Daily Summary

Sent at `sleep_hour`:

```
📊 Daily Recap
Quizzed: 8 | Answered: 6 | Skipped: 1 | Timed out: 1
pentest: 82% | distributed-systems: 67%
Tomorrow: ~7 questions
Streak: 15 days
```

### Config

```yaml
# _meta/quiz-delivery.yml
wake_hour: 8
sleep_hour: 22
min_gap_minutes: 30
max_gap_minutes: 180
max_questions_per_day: 12
timeout_hours: 4
```
