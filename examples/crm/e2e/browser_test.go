package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// macChrome is where Google Chrome lives on macOS.
const macChrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// chromePath finds the browser: CRM_CHROME, else Google Chrome on macOS,
// else chromium or google-chrome on PATH. A missing browser is an error,
// never a skip.
func chromePath() (string, error) {
	if p := os.Getenv("CRM_CHROME"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("CRM_CHROME names no browser: %v", err)
		}
		return p, nil
	}
	if _, err := os.Stat(macChrome); err == nil {
		return macChrome, nil
	}
	for _, name := range []string{"chromium", "google-chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no Chrome or Chromium found (tried " + macChrome +
		", chromium and google-chrome on PATH): set CRM_CHROME to the path of a Chrome or Chromium binary")
}

// page is one tab of a headless Chrome that belongs to one test. The
// browser is closed, and its profile removed, when the test ends.
type page struct {
	t   *testing.T
	ctx context.Context
}

func openBrowser(t *testing.T) *page {
	t.Helper()
	path, err := chromePath()
	if err != nil {
		t.Fatal(err)
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(path), chromedp.UserDataDir(t.TempDir()))
	actx, acancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(actx)
	t.Cleanup(func() {
		_ = chromedp.Cancel(ctx)
		cancel()
		acancel() // kills the browser if it still runs and waits for it
	})
	if err := chromedp.Do(ctx); err != nil {
		t.Fatalf("Chrome at %s did not start: %v (set CRM_CHROME to another Chrome or Chromium binary)", path, err)
	}
	return &page{t: t, ctx: ctx}
}

func (p *page) run(what string, steps ...chromedp.Action[chromedp.Void]) {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	if err := chromedp.Do(ctx, steps...); err != nil {
		p.t.Fatalf("browser: %s: %v", what, err)
	}
}

func byTestID(id string) chromedp.CSS { return chromedp.CSS(`[data-testid="` + id + `"]`) }

// open loads a page of the service under test.
func (p *page) open(path string) {
	p.t.Helper()
	p.run("open "+path, chromedp.Navigate(baseURL+path), chromedp.WaitReady(chromedp.CSS("body")))
}

func (p *page) fill(id, value string) {
	p.t.Helper()
	p.run("fill "+id, chromedp.SetValue(byTestID(id), value))
}

// click presses a button; the form it submits loads a new page, which the
// caller waits for with waitText.
func (p *page) click(id string) {
	p.t.Helper()
	p.run("click "+id, chromedp.Click(byTestID(id)))
}

// text reads the text of an element of the current page; ok is false when
// the page has no such element.
func (p *page) text(id string) (string, bool) {
	p.t.Helper()
	js := `(() => { const e = document.querySelector('[data-testid="` + id + `"]'); return e ? e.textContent.trim() : null })()`
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	s, err := chromedp.Run(ctx, chromedp.Evaluate[*string](js))
	if err != nil || s == nil {
		return "", false // no such element, or the page is loading; the caller polls
	}
	return *s, true
}

// waitText waits until the element's text satisfies match, while the page
// a click submitted loads.
func (p *page) waitText(id, want string, match func(got, want string) bool) string {
	p.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	last := "(no element)"
	for time.Now().Before(deadline) {
		if got, ok := p.text(id); ok {
			if match(got, want) {
				return got
			}
			last = got
		}
		time.Sleep(50 * time.Millisecond)
	}
	p.t.Fatalf("browser: %s is %q, want %q", id, last, want)
	return ""
}

func equal(got, want string) bool { return got == want }

func (p *page) waitEqual(id, want string) { p.t.Helper(); p.waitText(id, want, equal) }

func (p *page) waitContains(id, want string) string {
	p.t.Helper()
	return p.waitText(id, want, strings.Contains)
}

// absent fails when the current page has the element.
func (p *page) absent(id string) {
	p.t.Helper()
	if got, ok := p.text(id); ok {
		p.t.Fatalf("browser: %s is shown (%q), want no such element", id, got)
	}
}

// selectUser picks a user in the switcher and waits until the page acts as
// that user.
func (p *page) selectUser(id, email string) {
	p.t.Helper()
	p.run("select user", chromedp.SetValue(byTestID("user-select"), id))
	p.click("user-use")
	p.waitEqual("user-current", email)
}
