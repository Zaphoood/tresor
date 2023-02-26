# Tresor 🗝️

A [KeePass](https://www.google.com/search?channel=fs&client=ubuntu&q=keepass) TUI written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Bubbles](https://github.com/charmbracelet/bubbles), featuring a [ranger](https://github.com/ranger/ranger)-inspired layout and vi-like keybindings.

<p align="center">
  <br>
  <img src="./demo/demo.gif" width="90%"/>
</p>
<br>

__⚠️ Disclaimer ⚠️__ I am not a cryptography expert. I cannot guarantee that this application is secure. Use at your own risk.

## Getting started

First, install [Go](https://go.dev/) if you haven't already.

To install `tresor` to your GOPATH, simply run
```
go install ./cmd/tresor
```

If you just want to build the application, run
```
go build ./cmd/tresor
```
This will save the executable to the working directory.

## Usage

To open a file, run `tresor FILE`. Alternatively, run just `tresor` and input the filename when prompted.

After inputting your password, the KeePass database should open.
Navigate using the <kbd>H</kbd>, <kbd>J</kbd>, <kbd>K</kbd> and <kbd>L</kbd> keys, press <kbd>Ctrl-C</kbd> to exit.

When hovering over an entry, press <kbd>Enter</kbd> to copy its password to the system clipboard. The clipboard will be cleared automatically after ten seconds.

## Roadmap

This project is at a very early stage. Here is the short list of what is possible right now:

 * Open and close `.kdbx` files of version 3.1
 * Navigate groups and entries
 * Preview entries
 * Copy passwords to clipboard

Some features planned for the future are:

 * Searching through and editing entries
 * Creating new databases
 * Customization via config file (inculding color scheme)
 * Kdbx version 4 support
 * Auto-type
 * WebDAV integration (and possibly other file sharing protocols)

The main goal is to implement all the core features provided by [KeePass](https://keepass.info/).

Note that currently, only Linux is actively supported, but the code should compile on Windows and macOS as well. Try it out &mdash; it might just work ;)

## Contributing

If you find any bugs or have a feature request, please file an issue.
Pull requests are also welcome :)
