package prompt

type Options struct {
	SizeLimit int
}

type Result struct {
	Content   string
	Truncated bool
}
