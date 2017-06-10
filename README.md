# Extract

Package `extract` provides a faculty to extract content from HTML code.

## Install

To install, use `go get`:

```bash
$ go get -u github.com/elpinal/extract
```

-u flag stands for "update".

## Examples

```go
src := "<head> <title> I am a Gopher </title> </head> <p>This is a content</p>"
title, content, err := extract.Extract(strings.NewReader(src))
```

## Contribution

1. Fork ([https://github.com/elpinal/extract/fork](https://github.com/elpinal/extract/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create a new Pull Request

## Author

[elpinal](https://github.com/elpinal)
