# Tresor 🗝️

| This project is no longer actively maintained.

Tresor (pronounced [tʁeˈzoːɐ̯] &mdash; or any way you like) is a [KeePass](https://keepass.info/) TUI written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Bubbles](https://github.com/charmbracelet/bubbles), featuring a [ranger](https://github.com/ranger/ranger)-inspired layout and vi-like keybindings.

<p align="center">
  <br>
  <img src="./demo/demo.gif" width="90%"/>
</p>
<br>

**⚠️ Disclaimer ⚠️** I am not a cryptography expert. I cannot guarantee that this application is secure. Use at your own risk.

## Getting started

First, install [Go](https://go.dev/) if you haven't already.

To install tresor to your `GOPATH`, simply run

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
After entering your password, the KeePass database should open.

Navigate using the `h`, `j`, `k` and `l` keys, type `:q` and hit `Enter` to exit.

### Key bindings

**Quick reference:**

| Key       | Action                                                  |
| --------- | ------------------------------------------------------- |
| `j` / `k` | Go up / down                                            |
| `h`       | Leave group or entry                                    |
| `l`       | Enter focused group or entry                            |
| `y`       | Copy password of focused entry or field value           |
| `d`       | Delete focused field value                              |
| `u`       | Undo last change                                        |
| `C-r`     | Redo change                                             |
| `c`       | Change title of focused entry or value of focused field |

When hovering over an entry, press `y` to copy its password to the system clipboard.
After focusing an entry with `h` or `Enter`, you can select individual fields and copy their value (also using `y`).
For encrypted values, the clipboard will be cleared automatically after ten seconds.

To delete a field, select it and press `d`. Some fields (i. e. title, username and password) are special: They won't be
deleted, instead their value will be set to an empty string. This is because KeePass considers these fields 'default
fields', and they should always be present for any given entry.

Note that deleting entries or groups is not implemented yet.

Entries can be renamed by pressing `c` and entering a new name. As with `y`, you can also change individual fields by
selecting them and pressing `c`.

To undo any change, press `u`. To redo, press `C-r`.

### Searching

To search through the current group, type `/` (or `?` for backward search) followed by a query, and press `Enter`.
Cycle through matches using `n` and `N` for the next and previous match, respectively.

### Commands

Commands work just like in vim: To execute a command, type `:` followed by the name of the command and optionally some
arguments; then, press `Enter`.
These commands are currently available:

| Command               | Action                                                                  |
| --------------------- | ----------------------------------------------------------------------- |
| `:q`                  | Quit without saving                                                     |
| `:w`                  | Save file. Specify path with `:w <file>`                                |
| `:wq`, `:x`           | Save file and quit. Specifying the path works analogous to `:w`         |
| `:e`                  | Reload current file (You will be prompted to enter your password again) |
| `:e <file>`           | Load `<file>` from disk                                                 |
| `:change <new-value>` | Set value of focused entry / field to `<new-value>` (shortcut: `c`)     |

Note that currently, the `:w` command is pretty much useless, since editing entries is not supported, so it's not possible to actually make changes to a file. However, the last selected group is, in fact, stored and remembered when re-opening.
