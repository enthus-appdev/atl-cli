package iostreams

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestSystem tests that System returns a properly configured IOStreams.
func TestSystem(t *testing.T) {
	ios := System()

	if ios == nil {
		t.Fatal("System() returned nil")
	}
	if ios.In == nil {
		t.Error("System().In should not be nil")
	}
	if ios.Out == nil {
		t.Error("System().Out should not be nil")
	}
	if ios.ErrOut == nil {
		t.Error("System().ErrOut should not be nil")
	}
}

// TestTest tests that Test returns IOStreams suitable for testing.
func TestTest(t *testing.T) {
	ios := Test()

	if ios == nil {
		t.Fatal("Test() returned nil")
	}

	// Test streams should not be terminals
	if ios.IsStdinTTY {
		t.Error("Test().IsStdinTTY should be false")
	}
	if ios.IsStdoutTTY {
		t.Error("Test().IsStdoutTTY should be false")
	}
	if ios.IsStderrTTY {
		t.Error("Test().IsStderrTTY should be false")
	}

	// Color should be disabled for tests
	if ios.ColorEnabled() {
		t.Error("Test().ColorEnabled() should be false")
	}
}

// TestColorEnabled tests the ColorEnabled getter.
func TestColorEnabled(t *testing.T) {
	ios := &IOStreams{colorEnabled: true}
	if !ios.ColorEnabled() {
		t.Error("ColorEnabled() should return true when colorEnabled is true")
	}

	ios.colorEnabled = false
	if ios.ColorEnabled() {
		t.Error("ColorEnabled() should return false when colorEnabled is false")
	}
}

// TestSetColorEnabled tests the SetColorEnabled setter.
func TestSetColorEnabled(t *testing.T) {
	ios := &IOStreams{}

	ios.SetColorEnabled(true)
	if !ios.colorEnabled {
		t.Error("SetColorEnabled(true) should set colorEnabled to true")
	}

	ios.SetColorEnabled(false)
	if ios.colorEnabled {
		t.Error("SetColorEnabled(false) should set colorEnabled to false")
	}
}

// TestNullReader tests that nullReader returns EOF immediately.
func TestNullReader(t *testing.T) {
	r := &nullReader{}
	buf := make([]byte, 10)

	n, err := r.Read(buf)
	if n != 0 {
		t.Errorf("nullReader.Read() returned n=%d, want 0", n)
	}
	if err != io.EOF {
		t.Errorf("nullReader.Read() returned err=%v, want io.EOF", err)
	}
}

// TestIOStreamsWithCustomStreams tests using IOStreams with custom streams.
func TestIOStreamsWithCustomStreams(t *testing.T) {
	inBuf := strings.NewReader("input data")
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	ios := &IOStreams{
		In:           inBuf,
		Out:          outBuf,
		ErrOut:       errBuf,
		IsStdinTTY:   false,
		IsStdoutTTY:  false,
		IsStderrTTY:  false,
		colorEnabled: false,
	}

	// Test writing to Out
	_, err := ios.Out.Write([]byte("test output"))
	if err != nil {
		t.Errorf("Write to Out failed: %v", err)
	}
	if outBuf.String() != "test output" {
		t.Errorf("Out buffer = %q, want %q", outBuf.String(), "test output")
	}

	// Test writing to ErrOut
	_, err = ios.ErrOut.Write([]byte("test error"))
	if err != nil {
		t.Errorf("Write to ErrOut failed: %v", err)
	}
	if errBuf.String() != "test error" {
		t.Errorf("ErrOut buffer = %q, want %q", errBuf.String(), "test error")
	}

	// Test reading from In
	buf := make([]byte, 5)
	n, err := ios.In.Read(buf)
	if err != nil {
		t.Errorf("Read from In failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Read n=%d, want 5", n)
	}
	if string(buf) != "input" {
		t.Errorf("Read data = %q, want %q", string(buf), "input")
	}
}

// TestIOStreamsForTesting is a helper pattern for creating test IOStreams
// with accessible buffers for verification.
func TestIOStreamsForTesting(t *testing.T) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	ios := &IOStreams{
		In:           &nullReader{},
		Out:          outBuf,
		ErrOut:       errBuf,
		IsStdinTTY:   false,
		IsStdoutTTY:  false,
		IsStderrTTY:  false,
		colorEnabled: false,
	}

	// Simulate command output
	ios.Out.Write([]byte("Success!\n"))
	ios.ErrOut.Write([]byte("Warning: something\n"))

	// Verify output
	if !strings.Contains(outBuf.String(), "Success!") {
		t.Error("Expected 'Success!' in output buffer")
	}
	if !strings.Contains(errBuf.String(), "Warning:") {
		t.Error("Expected 'Warning:' in error buffer")
	}
}
