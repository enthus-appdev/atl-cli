package iostreams

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// IOStreams provides access to standard input, output, and error streams.
// It abstracts the I/O for easier testing and flexibility.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	// IsStdinTTY indicates if stdin is a terminal
	IsStdinTTY bool
	// IsStdoutTTY indicates if stdout is a terminal
	IsStdoutTTY bool
	// IsStderrTTY indicates if stderr is a terminal
	IsStderrTTY bool

	// colorEnabled indicates if colored output should be used
	colorEnabled bool
}

// System returns IOStreams connected to the system's standard streams.
func System() *IOStreams {
	stdoutIsTTY := isTerminal(os.Stdout)
	stderrIsTTY := isTerminal(os.Stderr)

	ios := &IOStreams{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		IsStdinTTY:  isTerminal(os.Stdin),
		IsStdoutTTY: stdoutIsTTY,
		IsStderrTTY: stderrIsTTY,
	}

	// Enable color by default if stdout is a TTY and NO_COLOR is not set
	ios.colorEnabled = stdoutIsTTY && os.Getenv("NO_COLOR") == ""

	return ios
}

// Test returns IOStreams suitable for testing.
func Test() *IOStreams {
	return &IOStreams{
		In:           &nullReader{},
		Out:          io.Discard,
		ErrOut:       io.Discard,
		IsStdinTTY:   false,
		IsStdoutTTY:  false,
		IsStderrTTY:  false,
		colorEnabled: false,
	}
}

// ColorEnabled returns true if colored output should be used.
func (ios *IOStreams) ColorEnabled() bool {
	return ios.colorEnabled
}

// SetColorEnabled sets whether colored output should be used.
func (ios *IOStreams) SetColorEnabled(enabled bool) {
	ios.colorEnabled = enabled
}

// isTerminal checks if a file is a terminal.
func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// nullReader is an io.Reader that always returns EOF.
type nullReader struct{}

func (r *nullReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
