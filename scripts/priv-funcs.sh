#!/usr/bin/env bash
#
# Run this horrifying little hack when you've changed the privilege list and
# don't remember what you have or have not put into the template functions
# list. I do it like this:
#
#   ./scripts/priv-funcs.sh >> src/cmd/server/internal/responder/templates.go
#
# Then I edit templates.go, replacing the old privilege functions with what's
# at the end of the file. Run a quick GoFmt command and done!

cat src/privilege/privilege.go | grep '=\s*newPrivilege' | sed 's|^\s*\([a-zA-Z_]\+\)\s*=\s*newPriv.*$|"\1": func() *privilege.Privilege { return privilege.\1 },|'

