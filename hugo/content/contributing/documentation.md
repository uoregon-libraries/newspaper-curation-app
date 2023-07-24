---
title: Contributing to Documentation
weight: 30
description: How to help with NCA's docs
---

The documentation for this site is produce using [Hugo](https://gohugo.io/) and
the [TechDoc](https://themes.gohugo.io/hugo-theme-techdoc/) theme.

We want our documentation to help you use NCA (okay, again, if I'm being
honest, this is really just for UO to remember how to use our app), and as such
the documentation itself needs to be easy to edit and keep up-to-date.

## Starting out

Get Hugo installed.  It's a trivial standalone application that can be
installed with minimal fuss:
[installing Hugo](https://gohugo.io/getting-started/installing).

Once you have it, you can download the NCA codebase from Github and start
editing.  Change to the `hugo` subdirectory in NCA, and you'll see all the
content under `content/`.

Normally you could use `hugo serve` to test out the documentation. But due to a
bug in the theme we chose, documentation generation is inconsistent and doesn't
actually do what it should every time.

The solution: `make docserver`. This uses whatever is in `docs`, but copies it
into a `localdocs` directory, replaces URLs to work locally, and fires up an
Apache web server listening on localhost:8080. This ensures that what you're
testing locally is in fact whatever you generated most recently in `docs`.

## Magic / Rules

Everything under `hugo/content` abides by certain rules.  These rules are few,
but can be very confusing at first.  If you know Hugo pretty well already, this
probably makes sense to you, but I'm just using it for the first time ever,
so....

**Rule 1**: `hugo/content/` seems to be a "special" location.  Nothing there shows
up in the site navigation area except `_index.md`, which is our main navigation
landing page.  If you created a file called `hugo/content/foo.md`, you could
reach it by visiting `http://localhost:1313/newspaper-curation-app/foo`, but it
won't show up in the navigation.

**Rule 2**: Everything else has to live under a subdirectory, period.  If you
want it in the navigation, and you almost always do, you need it under a
subdirectory, such as this document which lives under `contributing`.

**Rule 3**: The "home" page of a subdirectory is always called `index.md`.
That file is magic and shows up it the hierarchy of pages at one level above
the subdirectory.  e.g., `hugo/content/contributing/_index.md` is at the root
level of the navigation menu, and everything else in
`hugo/content/contributing/` is shown in the navigation menu as being nested
under that.

**Rule 4**: All pages need to be in markdown format and need "frontmatter".
Example frontmatter can be seen on any of these pages at the top, between two
lines which are simply "---".  The title is critical, as it shows up in the
navbar, the site's `<title>`, and is the first heading (`<h1>`) on the page.
The weight is where the page shows up in the navigation bar *relative to the
structure in which the page lives*.  This can be confusing, so...

**Rule 5**: A page's "weight" is confusing because the hierarchy is confusing.
All pages are siblings of one another if they're in the same directory *except*
`_index.md`.  That magical, special page is actually a *parent* of the other
pages in that directory.  This means the weight listed in `foo/one.md` is how
"one" will be ordered compared to `foo/two.md`.  But `foo/bar/_index.md` will
also be ordered relative to those two pages, because it is the *parent* of
`foo/bar/*`, which makes it the **sibling** of `foo/*`.  Confusing, right?

## Guidelines

This is more straightforward.  YAY!

- Never include a level one heading, e.g., `# Title`.  Hugo generates a level
  one heading based on the title, so this is unnecessary, and can be an
  accessibility problem.
- Use hashes, not underscores, for level two headings.  It's just easier to
  grep for structural elements this way.
- Be explicit about weights, even though it is confusing and sucky.
- Use 80-character width for documentation unless it splits up a link's text
  (that messes up my markdown editor in some cases).  80 characters.  I use
  tmux with multiple panes, and 80 characters will never annoy me.  If you
  contribute something annoying, it may or may not ever get into the repo.
  Call me old if you like, but I'm king where this repository is concerned.

## Linking to other pages

If you want to link to another document, use a relative path, not a full URL,
because a full URL will either work on the live site (e.g.,
https://foo.github.io/blah) or locally (http://localhost:1313/blah), but not
both.

An easy way to do this is just start with a slash and append everything that
comes after `hugo/content/` minus the ".md".  e.g. you can do this:

[docs](/contributing/documentation)

with this code: `[docs](/contributing/documentation)`.
