package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// node wraps the chromedp node to give us a persistent element we can interact
// with in small ways
type node struct {
	b    *browser
	ctx  context.Context
	node *cdp.Node
}

func (n *node) click() {
	var err = chromedp.MouseClickNode(n.node).Do(n.ctx)
	if err != nil {
		n.b.fatalf("Unable to click node: %s", err)
	}
}

func (n *node) focus() {
	var err = dom.Focus().WithNodeID(n.node.NodeID).Do(n.ctx)
	if err != nil {
		n.b.fatalf("Unable to focus node: %s", err)
	}
}

func (n *node) blur() {
	var err = runJS(n, "function() { this.blur(); }", nil, nil)
	if err != nil {
		n.b.fatalf("Unable to blur node: %s", err)
	}
}

func (n *node) keyEvent(text string, opts ...func(*input.DispatchKeyEventParams) *input.DispatchKeyEventParams) {
	var err = chromedp.KeyEvent(text, opts...).Do(n.ctx)
	if err != nil {
		n.b.fatalf("Unable to enter key sequence %q: %s", text, err)
	}
}

func (n *node) deleteText() {
	n.focus()
	n.keyEvent("a", chromedp.KeyModifiers(input.ModifierCtrl))
	n.keyEvent("\b")
}

func (n *node) typeText(text string) {
	n.focus()
	n.keyEvent(text)
}

func (n *node) innerText() string {
	return n.getJSAttribute("innerText")
}

func (n *node) innerHTML() string {
	return n.getJSAttribute("innerHTML")
}

func (n *node) outerHTML() string {
	return n.getJSAttribute("outerHTML")
}

func (n *node) getJSAttribute(attr string) string {
	var fn = fmt.Sprintf("function() { return this.%s; }", attr)
	var val string
	var err = runJS(n, fn, nil, &val)
	if err != nil {
		n.b.fatalf("Unable to call JS to get attribute %q: %s", attr, err)
	}
	return val
}

// runJS is a near-exact copy of a chromedp *internal* function. Why is it
// internal? I guess the same reason chromedp's API is so backward to begin
// with: reasons.
func runJS(n *node, function string, args []any, result any) error {
	var r, err = dom.ResolveNode().WithNodeID(n.node.NodeID).Do(n.ctx)
	if err != nil {
		return err
	}
	var callfn = func(p *runtime.CallFunctionOnParams) *runtime.CallFunctionOnParams {
		return p.WithObjectID(r.ObjectID)
	}
	err = chromedp.CallFunctionOn(function, result, callfn, args...).Do(n.ctx)

	if err != nil {
		return err
	}

	// Try to release the remote object.
	// It will fail if the page is navigated or closed,
	// and it's okay to ignore the error in this case.
	_ = runtime.ReleaseObject(r.ObjectID).Do(n.ctx)

	return nil
}

func (n *node) find(selector string) *node {
	return n.b.find(selector, chromedp.FromNode(n.node))
}

func (n *node) findAll(selector string) []*node {
	return n.b.findAll(selector, chromedp.FromNode(n.node))
}

func (n *node) assertHasText(text string) {
	if !strings.Contains(n.innerText(), text) {
		n.b.t.Logf("n.innerText: %q", n.innerText())
		n.b.fatalf("Expected node %#v to have %q in its text, but it didn't", n, text)
	}
}

func (n *node) assertHasHTML(text string) {
	if !strings.Contains(n.innerHTML(), text) {
		n.b.t.Logf("n.innerHTML: %q", n.innerHTML())
		n.b.fatalf("Expected node %#v to have %q in its HTML, but it didn't", n, text)
	}
}

func (n *node) assertNoText(text string) {
	if strings.Contains(n.innerText(), text) {
		n.b.t.Logf("n.innerText: %q", n.innerText())
		n.b.fatalf("Expected node %#v not to have %q in its text, but it did", n, text)
	}
}

func (n *node) assertNoHTML(text string) {
	if strings.Contains(n.innerHTML(), text) {
		n.b.t.Logf("n.innerHTML: %q", n.innerHTML())
		n.b.fatalf("Expected node %#v not to have %q in its HTML, but it did", n, text)
	}
}
