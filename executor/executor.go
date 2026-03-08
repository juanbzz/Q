package executor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/juanbzz/q"
)

// CommandValidator checks if a command is safe to execute.
type CommandValidator interface {
	Validate(command string) error
}

// BlocklistValidator blocks commands matching dangerous patterns.
type BlocklistValidator struct {
	patterns []*regexp.Regexp
}

var DefaultBlockedPatterns = []string{
	`rm\s+-[rf]*\s+/`,
	`rm\s+-[rf]*\s+\*`,
	`rm\s+-[rf]*\s+~`,
	`>\s*/dev/sd`,
	`mkfs`,
	`dd\s+if=.*/dev/`,
	`dd\s+of=.*/dev/`,
	`chmod\s+777\s+/`,
	`chown\s+-R\s+.*\s+/`,
	`curl.*\|\s*(ba)?sh`,
	`wget.*\|\s*(ba)?sh`,
	`:\(\)\{\s*:\|:&\s*\};:`,
	`/dev/null\s*>\s*/etc/`,
	`>\s*/etc/passwd`,
	`>\s*/etc/shadow`,
	`shutdown`,
	`reboot`,
	`init\s+0`,
	`halt`,
	`poweroff`,
}

func NewBlocklistValidator(patterns []string) (*BlocklistValidator, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &BlocklistValidator{patterns: compiled}, nil
}

func NewDefaultBlocklistValidator() *BlocklistValidator {
	v, _ := NewBlocklistValidator(DefaultBlockedPatterns)
	return v
}

func (v *BlocklistValidator) Validate(command string) error {
	for _, re := range v.patterns {
		if re.MatchString(command) {
			return &q.ProcessErr{
				Type:    q.ProcessErrExecution,
				Message: fmt.Sprintf("Command blocked for safety: matches pattern %q. Please use a safer alternative.", re.String()),
			}
		}
	}
	return nil
}

// BashParser extracts bash commands from markdown code blocks.
// Implements q.Parser.
type BashParser struct{}

var commandRegex = regexp.MustCompile("(?s)```bash\\s*\\n(.*?)\\n```")

func NewBashParser() *BashParser {
	return &BashParser{}
}

func (p *BashParser) ParseAction(response string) (q.Action, error) {
	matches := commandRegex.FindAllStringSubmatch(response, -1)

	if len(matches) == 0 {
		return q.Action{}, &q.ProcessErr{
			Type:    q.ProcessErrFormat,
			Message: "No bash command found. If the task is complete, respond with TASK_COMPLETE. Otherwise, provide exactly one command in ```bash``` block.",
		}
	}

	if len(matches) > 1 {
		return q.Action{}, &q.ProcessErr{
			Type:    q.ProcessErrFormat,
			Message: fmt.Sprintf("Found %d commands, expected exactly one. Please provide a single command in ```bash``` block.", len(matches)),
		}
	}

	command := strings.TrimSpace(matches[0][1])
	if command == "" {
		return q.Action{}, &q.ProcessErr{
			Type:    q.ProcessErrFormat,
			Message: "Empty command in bash block. Please provide a valid command.",
		}
	}

	return q.Action{
		Type:    q.ActionTypeBash,
		Command: command,
	}, nil
}
