package prompt

import "github.com/VemorPhose/TailChase/internal/project"

type Options struct {
	SizeLimit      int
	Delta          bool
	AttemptHistory project.AttemptHistory
}

type Result struct {
	Content   string
	Truncated bool
}
