package native

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Terminal handles raw terminal I/O without external dependencies
type Terminal struct {
	in  *os.File
	out io.Writer
	mu  sync.Mutex

	width, height int
	oldState      *term.State
	running       bool

	// Buffer for efficient rendering
	buffer []byte
}

func New(out io.Writer) (*Terminal, error) {
	in, err := os.Open("/dev/tty")
	if err != nil {
		// Fallback for Windows or when /dev/tty unavailable
		in = os.Stdin
	}

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 80, 24 // Default fallback
	}

	t := &Terminal{
		in:     in,
		out:    out,
		width:  w,
		height: h,
		buffer: make([]byte, 0, 4096),
	}

	return t, nil
}

// EnableRaw sets terminal to raw mode (no buffering, no echo)
func (t *Terminal) EnableRaw() error {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}
	t.oldState = state
	t.running = true
	return nil
}

// DisableRaw restores terminal to original state
func (t *Terminal) DisableRaw() error {
	if t.oldState != nil {
		return term.Restore(int(os.Stdin.Fd()), t.oldState)
	}
	return nil
}

// Fast escape sequences (minimal allocations)
func (t *Terminal) ClearScreen() {
	t.out.Write([]byte("\x1b[2J"))
	t.out.Write([]byte("\x1b[H"))
}

func (t *Terminal) MoveCursor(x, y int) {
	fmt.Fprintf(t.out, "\x1b[%d;%dH", y+1, x+1)
}

func (t *Terminal) HideCursor() {
	t.out.Write([]byte("\x1b[?25l"))
}

func (t *Terminal) ShowCursor() {
	t.out.Write([]byte("\x1b[?25h"))
}

// Color codes (256-color palette)
func (t *Terminal) SetColor(fg, bg int) {
	if fg >= 0 {
		fmt.Fprintf(t.out, "\x1b[38;5;%dm", fg)
	}
	if bg >= 0 {
		fmt.Fprintf(t.out, "\x1b[48;5;%dm", bg)
	}
}

func (t *Terminal) SetBold() {
	t.out.Write([]byte("\x1b[1m"))
}

func (t *Terminal) SetDim() {
	t.out.Write([]byte("\x1b[2m"))
}

func (t *Terminal) ResetColor() {
	t.out.Write([]byte("\x1b[0m"))
}

// Write text to terminal
func (t *Terminal) Write(text string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, err := t.out.Write([]byte(text))
	return err
}

// Flush writes buffer to stdout
func (t *Terminal) Flush() error {
	if f, ok := t.out.(*os.File); ok {
		return f.Sync()
	}
	return nil
}

// Size returns terminal width and height
func (t *Terminal) Size() (int, int) {
	return t.width, t.height
}

// UpdateSize polls terminal size (call on WindowSizeMsg)
func (t *Terminal) UpdateSize() {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	if w > 0 && h > 0 {
		t.width = w
		t.height = h
	}
}

// GetKey reads a single key non-blocking
func (t *Terminal) GetKey() (rune, error) {
	if t.in == nil {
		return 0, fmt.Errorf("input not available")
	}
	var buf [1]byte
	n, err := t.in.Read(buf[:])
	if n > 0 {
		return rune(buf[0]), nil
	}
	return 0, err
}

// WaitTimeout waits for input with timeout
func (t *Terminal) WaitTimeout(d time.Duration) (rune, error) {
	done := make(chan rune)
	go func() {
		key, _ := t.GetKey()
		done <- key
	}()

	select {
	case key := <-done:
		return key, nil
	case <-time.After(d):
		return 0, fmt.Errorf("timeout")
	}
}

func (t *Terminal) Close() error {
	t.DisableRaw()
	if t.in != os.Stdin && t.in != nil {
		return t.in.Close()
	}
	return nil
}
