# Tresor 🗝️

A [KeePass](https://keepass.info/) TUI written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Bubbles](https://github.com/charmbracelet/bubbles), featuring a [ranger](https://github.com/ranger/ranger)-inspired layout and vi-like keybindings.

<p align="center">
  <br>
  <img src="./demo/demo.gif" width="90%"/>
</p>
<br>

**⚠️ Disclaimer ⚠️** I am not a cryptography expert. I cannot guarantee that this application is secure. Use at your own risk.

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

To open a file, run `tresor <file>`. Alternatively, run just `tresor` and input the filename when prompted.

After inputting your password, the KeePass database should open.
Navigate using the `h`, `j`, `k` and `l` keys, type `:q` and hit <kbd>Enter</kbd> to exit.

When hovering over an entry, press `y` to copy its password to the system clipboard.
After focusing an entry with `h` or <kbd>Enter</kbd>, you can select and copy specific fields (again using `y`).
For encrypted values, the clipboard will be cleared automatically after ten seconds.

Commands work just like in vim: To execute a command, type `:` followed by the command and press <kbd>Enter</kbd>.
These commands are currently available:

| Command     | Action                                                                  |
| ----------- | ----------------------------------------------------------------------- |
| `:q`        | Quit without saving                                                     |
| `:w`        | Save file. Specify path with `:w <file>`                                |
| `:wq`, `:x` | Save file and quit. Specifying the path works analogous to `:w`         |
| `:e`        | Reload current file (You will be prompted to enter your password again) |
| `:e <file>` | Load `<file>` from disk                                                 |

Note that currently, the `:w` command is pretty much useless, since editing entries is not supported, so it's not possible to actually make changes to a file. However, the last selected group is, in fact, stored and remembered when re-opening.

To search through the current group, type `/` (or `?` for backward search), followed by a query and press <kbd>Enter</kbd>.
Cycle through search results using `n` and `N`.

## Roadmap

This project is at a very early stage. Here is the short list of what is possible right now:

- Load and save `.kdbx` files of version 3.1
- Navigate groups and entries
- Preview entries
- Copy passwords to clipboard

Some features planned for the future are:

- Editing entries
- Fuzzy-finding
- Creating new databases
- Customization via config file (inculding color scheme)
- Kdbx version 4 support
- Auto-type
- WebDAV integration (and possibly other file sharing protocols)

The main goal is to implement all the core features provided by [KeePass](https://keepass.info/).

Note that currently, only Linux is actively supported, but the code should compile on Windows and macOS as well. Try it out &mdash; it might just work ;)

## Contributing

If you find any bugs or have a feature request, please file an issue.
Pull requests are also welcome :)
