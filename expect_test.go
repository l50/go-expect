// Copyright 2018 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expect

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	ErrWrongAnswer = errors.New("wrong answer")
)

type Survey struct {
	Prompt string
	Answer string
}

func Prompt(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for _, survey := range []Survey{
		{
			"What is 1+1?", "2",
		},
		{
			"What is Netflix backwards?", "xilfteN",
		},
	} {
		fmt.Fprintf(out, "%s: ", survey.Prompt)
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		fmt.Fprint(out, text)
		text = strings.TrimSpace(text)
		if text != survey.Answer {
			return ErrWrongAnswer
		}
	}

	return nil
}

func newTestConsole(t *testing.T, opts ...ConsoleOpt) (*Console, error) {
	opts = append([]ConsoleOpt{
		expectNoError(t),
		sendNoError(t),
		WithDefaultTimeout(time.Second),
	}, opts...)
	return NewTestConsole(t, opts...)
}

func expectNoError(t *testing.T) ConsoleOpt {
	return WithExpectObserver(
		func(matchers []Matcher, buf string, err error) {
			if err == nil {
				return
			}
			if len(matchers) == 0 {
				t.Fatalf("Error occurred while matching %q: %s\n%s", buf, err, string(debug.Stack()))
			} else {
				var criteria []string
				for _, matcher := range matchers {
					criteria = append(criteria, fmt.Sprintf("%q", matcher.Criteria()))
				}
				t.Fatalf("Failed to find [%s] in %q: %s\n%s", strings.Join(criteria, ", "), buf, err, string(debug.Stack()))
			}
		},
	)
}

func sendNoError(t *testing.T) ConsoleOpt {
	return WithSendObserver(
		func(msg string, n int, err error) {
			if err != nil {
				t.Fatalf("Failed to send %q: %s\n%s", msg, err, string(debug.Stack()))
			}
			if len(msg) != n {
				t.Fatalf("Only sent %d of %d bytes for %q\n%s", n, len(msg), msg, string(debug.Stack()))
			}
		},
	)
}

func testCloser(t *testing.T, closer io.Closer) {
	if err := closer.Close(); err != nil {
		t.Errorf("Close failed: %s", err)
		debug.PrintStack()
	}
}

func TestExpectf(t *testing.T) {
	t.Parallel()

	c, err := newTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.Expectf("What is 1+%d?", 1)
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.SendLine("2")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.Expectf("What is %s backwards?", "Netflix")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.SendLine("xilfteN")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.ExpectEOF()
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestExpect(t *testing.T) {
	t.Parallel()

	c, err := newTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.ExpectString("What is 1+1?")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.SendLine("2")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.ExpectString("What is Netflix backwards?")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.SendLine("xilfteN")
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		_, err = c.ExpectEOF()
		if err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	// close the pts so we can expect EOF
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestExpectOutput(t *testing.T) {
	t.Parallel()

	c, err := newTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := c.ExpectString("What is 1+1?"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c.SendLine("3"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c.ExpectEOF(); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	err = Prompt(c.Tty(), c.Tty())
	if err == nil || err != ErrWrongAnswer {
		t.Errorf("Expected error '%s' but got '%s' instead", ErrWrongAnswer, err)
	}
	testCloser(t, c.Tty())
	wg.Wait()
}

func TestExpectDefaultTimeout(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t, WithDefaultTimeout(0))
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := Prompt(c.Tty(), c.Tty())
		if err != nil && !strings.Contains(err.Error(), "file already closed") {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Add a short sleep to ensure Prompt is running before ExpectString
		time.Sleep(100 * time.Millisecond)

		_, err = c.ExpectString("What is 1+2?")
		if err == nil || !strings.Contains(err.Error(), "i/o timeout") {
			t.Errorf("Expected error to contain 'i/o timeout' but got '%s' instead", err)
		}
	}()

	<-done          // Wait for the ExpectString goroutine to complete
	c.Tty().Close() // Close the TTY to unblock the Prompt
	wg.Wait()       // Wait for the Prompt goroutine to finish
}

func TestExpectTimeout(t *testing.T) {
	t.Parallel()

	c, err := NewTestConsole(t)
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := Prompt(c.Tty(), c.Tty()); err != nil && !strings.Contains(err.Error(), "file already closed") {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Add a short sleep to ensure Prompt is running before Expect
		time.Sleep(100 * time.Millisecond)

		if _, err := c.Expect(String("What is 1+2?"), WithTimeout(0)); err == nil || !strings.Contains(err.Error(), "i/o timeout") {
			t.Errorf("Expected error to contain 'i/o timeout' but got '%s' instead", err)
		}
	}()

	<-done          // Wait for the Expect goroutine to complete
	c.Tty().Close() // Close the TTY to unblock the Prompt
	wg.Wait()       // Wait for the Prompt goroutine to finish
}

func TestExpectDefaultTimeoutOverride(t *testing.T) {
	t.Parallel()

	c, err := newTestConsole(t, WithDefaultTimeout(100*time.Millisecond))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := Prompt(c.Tty(), c.Tty()); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		time.Sleep(200 * time.Millisecond)
		c.Tty().Close()
	}()

	if _, err := c.ExpectString("What is 1+1?"); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if _, err := c.SendLine("2"); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if _, err := c.ExpectString("What is Netflix backwards?"); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if _, err := c.SendLine("xilfteN"); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if _, err := c.Expect(EOF, PTSClosed, WithTimeout(time.Second)); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	wg.Wait()
}

func TestConsoleChain(t *testing.T) {
	t.Parallel()

	c1, err := NewConsole(expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c1)

	var wg1 sync.WaitGroup
	wg1.Add(1)
	go func() {
		defer wg1.Done()
		if _, err := c1.ExpectString("What is Netflix backwards?"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c1.SendLine("xilfteN"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c1.ExpectEOF(); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	c2, err := newTestConsole(t, WithStdin(c1.Tty()), WithStdout(c1.Tty()))
	if err != nil {
		t.Errorf("Expected no error but got'%s'", err)
	}
	defer testCloser(t, c2)

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		if _, err := c2.ExpectString("What is 1+1?"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c2.SendLine("2"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c2.ExpectEOF(); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	if err := Prompt(c2.Tty(), c2.Tty()); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	testCloser(t, c2.Tty())
	wg2.Wait()

	testCloser(t, c1.Tty())
	wg1.Wait()
}

func TestEditor(t *testing.T) {
	if _, err := exec.LookPath("vi"); err != nil {
		t.Skip("vi not found in PATH")
	}
	t.Parallel()

	c, err := NewConsole(expectNoError(t), sendNoError(t))
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	defer testCloser(t, c)

	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	cmd := exec.Command("vi", file.Name())
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := c.Send("iHello world\x1b"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c.SendLine(":wq!"); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
		if _, err := c.ExpectEOF(); err != nil {
			t.Errorf("Expected no error but got '%s'", err)
		}
	}()

	if err := cmd.Run(); err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}

	testCloser(t, c.Tty())
	wg.Wait()

	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Errorf("Expected no error but got '%s'", err)
	}
	if string(data) != "Hello world\n" {
		t.Errorf("Expected '%s' to equal '%s'", string(data), "Hello world\n")
	}
}
