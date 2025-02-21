package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// browser defines a structure that holds all the bits we need in order to
// automate browser testing. This lets us call into a simple API, get fairly
// granular results, and not have to prefix everything with chromedp.Run(...).
//
// We violate the "never store context" rule (See
// https://go.dev/blog/context-and-structs for details), but this struct is
// test-only, doesn't have an exposed API, and doesn't have any functions where
// our use-case would benefit from per-call context passing.
type browser struct {
	t   *testing.T
	ctx context.Context
}

func newBrowser(t *testing.T) *browser {
	var c1, c2, c3 context.CancelFunc
	var ctx context.Context
	ctx, c1 = chromedp.NewRemoteAllocator(context.Background(), headlessURL)
	ctx, c2 = context.WithTimeout(ctx, time.Second*120)
	ctx, c3 = chromedp.NewContext(ctx)
	t.Cleanup(func() { c1(); c2(); c3() })

	var b = &browser{t: t, ctx: ctx}
	b.run(chromedp.EmulateViewport(1920, 4320), "setting initial viewport")
	return b
}

func (b *browser) exit(message string, screenshot bool) {
	b.t.Logf(message)

	var callers = stack()
	var lastCall = len(callers) - 1
	if screenshot {
		var shotname = fmt.Sprintf("/tmp/failure-%s.png", callers[lastCall].function)
		var err = b.writeScreenshot(shotname)
		if err != nil {
			b.t.Logf("Unable to save screenshot for failed action: %s", err)
		} else {
			b.t.Logf("Saved screenshot for failed action to %q", shotname)
		}
	}

	for i := lastCall; i >= 0; i-- {
		var f = callers[i]
		b.t.Logf("-- %s:%d: %s", f.file, f.line, f.function)
	}
	b.t.FailNow()
}

func (b *browser) fatalf(format string, args ...any) {
	b.exit(fmt.Sprintf(format, args...), true)
}

func (b *browser) run(action chromedp.Action, msg string) {
	var err = chromedp.Run(b.ctx, action)
	if err != nil {
		b.fatalf("Failure %s: %s", msg, err)
	}
}

func (b *browser) visit(url string) {
	b.run(chromedp.Navigate(url), fmt.Sprintf("visiting URL %q", url))
}

func (b *browser) followLink(pth string) {
	b.find(fmt.Sprintf("a[href$='/%s']", pth)).click()
}

func (b *browser) getLocation() string {
	var location string
	b.run(chromedp.Location(&location), "reading document location")
	return location
}

func (b *browser) wait(selector string) {
	b.run(chromedp.WaitVisible(selector, chromedp.ByQueryAll), fmt.Sprintf("waiting for %q to be visible", selector))
}

// getBody is a special version of "find" which ensures we wait for the node to
// show up, as it's generally only called after following a link, submitting a
// form, etc.
func (b *browser) getBody() *node {
	b.wait("body")
	return b.find("body")
}

func (b *browser) find(selector string, opts ...func(*chromedp.Selector)) *node {
	var nodes = b.findAll(selector, opts...)
	if len(nodes) == 0 {
		b.fatalf("No nodes found by selector %q", selector)
	}
	return nodes[0]
}

func (b *browser) findAll(selector string, opts ...func(*chromedp.Selector)) []*node {
	var nodes []*node
	var afterFn = func(ctx context.Context, _ runtime.ExecutionContextID, cnodes ...*cdp.Node) error {
		nodes = make([]*node, len(cnodes))
		for i, cnode := range cnodes {
			nodes[i] = &node{b: b, ctx: ctx, node: cnode}
		}
		return nil
	}

	var allOpts = append(opts, chromedp.ByQueryAll, chromedp.AtLeast(0))
	b.run(
		chromedp.QueryAfter(selector, afterFn, allOpts...),
		fmt.Sprintf("querying nodes by selector %q", selector),
	)

	return nodes
}

func (b *browser) writeScreenshot(filename string) error {
	// Call chromedp.Run, *not* browser.run, otherwise errors result in infinite
	// recursion as browser.run keeps calling writeScreenshot...
	var res []byte
	var err = chromedp.Run(b.ctx, screenAction(&res))
	if err != nil {
		return fmt.Errorf("Failure retrieving a screenshot: %s", err)
	}

	err = os.WriteFile(filename, res, 0644)
	if err != nil {
		return fmt.Errorf("unable to write full screenshot to file %q: %s", filename, err)
	}

	return nil
}

// screenAction is a copy of FullScreenshot from chromedp but with parameters
// that suck a bit less
func screenAction(res *[]byte) chromedp.EmulateAction {
	if res == nil {
		panic("res cannot be nil")
	}
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		*res, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}
