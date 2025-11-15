#!/usr/bin/env -S just -f

GO := "go"
PKG_IMPORT_PATH := "github.com/rafaelespinoza/slogtesting"

# list recipes
@default:
    just -f {{ justfile() }} --list --unsorted

# sanity check for compilation errors
build:
	{{ GO }} build {{ PKG_IMPORT_PATH }}/...

# get module dependencies, tidy them up
mod:
    {{ GO }} mod tidy

# run tests (override variable value ARGS to use test flags)
test *ARGS:
    {{ GO }} test {{ PKG_IMPORT_PATH }}/... {{ ARGS }}

# examine source code for suspicious constructs
vet *ARGS:
    {{ GO }} vet {{ ARGS }} {{ PKG_IMPORT_PATH }}/...
