package util

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiCyan  = "\033[36m"
	ansiGray  = "\033[90m"
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
)

type Colorizer struct {
	Enabled bool
}

func (c Colorizer) Header(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiBold + ansiCyan + s + ansiReset
}

func (c Colorizer) Key(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiBold + s + ansiReset
}

func (c Colorizer) Muted(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiGray + s + ansiReset
}

func (c Colorizer) Warn(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiRed + s + ansiReset
}

func (c Colorizer) Good(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiGreen + s + ansiReset
}
