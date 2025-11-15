slogtesting
---

[![](https://pkg.go.dev/badge/github.com/rafaelespinoza/slogtesting)](https://pkg.go.dev/github.com/rafaelespinoza/slogtesting)
[![tests](https://github.com/rafaelespinoza/slogtesting/actions/workflows/tests.yaml/badge.svg)](https://github.com/rafaelespinoza/slogtesting/actions/workflows/tests.yaml)

`slogtesting` is a golang library to test that your application's structured
logging outputs the intended data.
It requires the use of [log/slog](https://pkg.go.dev/log/slog).

It provides a `slog.Handler` implementation that captures `slog.Record` values
as the logger outputs them. Everything is done in-memory and there is no need
to parse a log entry to do a test. Instead, work with golang data structures
from the `log/slog` package.
