[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/rclone-tui)](https://goreportcard.com/report/github.com/darkhz/rclone-tui)

[![youtube](https://img.youtube.com/vi/Jmm55Jh5Nhc/1.jpg)](https://youtube.com/watch?v=Jmm55Jh5Nhc)

# rclone-tui
rclone-tui is a cross-platform manager for rclone. It aims to be on-par with the web GUI (--rc-web-gui) as well as provide more improvements and enhancements over its general functionality.

Click on the above thumbnail to watch the demo video.
## Features
- Monitor rclone stats via the dashboard
- Create, update, view and delete configurations
- Explore remotes and perform various operations
- Mount and unmount remotes
- View file transfer and progress information

## Installation
You can download the binaries present in the **Releases** page. <br /><br />
Alternatively, if the **go** compiler is present in your system, you can install it with the following command:<br />
`go install github.com/darkhz/rclone-tui@latest`

## Usage
```
rclone-tui [<flags>]

Flags:
--page       Load the specified page (one of dashboard, configuration, explorer, mounts).
--host       Specify a rclone host to connect to.
--password   Specify a login password.
--user       Specify a login username.
```

## Keybindings

### Application

#### Global
|Operation                   |Keybinding                  |
|----------------------------|----------------------------|
|Open job manager            |<kbd>Ctrl</kbd>+<kbd>j</kbd>|
|Show view switcher          |<kbd>Ctrl</kbd>+<kbd>n</kbd>|
|Cancel currently loading job|<kbd>Ctrl</kbd>+<kbd>x</kbd>|
|Suspend                     |<kbd>Ctrl</kbd>+<kbd>z</kbd>|
|Quit                        |<kbd>Ctrl</kbd>+<kbd>q</kbd>|

#### Configuration/Mounts only
|Operation                            |Keybinding                                       |
|-------------------------------------|-------------------------------------------------|
|Select button                        |<kbd>Enter</kbd>                                 |
|Move between buttons                 |<kbd>Ctrl</kbd>+<kbd>Left/Right</kbd>            |
|Move between sections (wizard only)  |<kbd>Shift</kbd>+<kbd>Tab</kbd>                  |
|Move between form items (wizard only)|<kbd>Ctrl</kbd>+<kbd>Down/Up</kbd>/<kbd>Tab</kbd>|
|Show form options                    |<kbd>Ctrl</kbd>+<kbd>o</kbd>                     |
|Toggle password display              |<kbd>Ctrl</kbd>+<kbd>p</kbd>                     |

### Configuration
#### Manager
|Operation |Keybinding  |
|----------|------------|
|Create new|<kbd>n</kbd>|
|Update    |<kbd>u</kbd>|
|Delete    |<kbd>d</kbd>|
|Filter    |<kbd>/</kbd>|

#### Wizard
|Operation     |Keybinding                  |
|--------------|----------------------------|
|Jump to option|<kbd>Ctrl</kbd>+<kbd>f</kbd>|
|Save          |<kbd>Ctrl</kbd>+<kbd>s</kbd>|
|Cancel        |<kbd>Ctrl</kbd>+<kbd>c</kbd>|

### Explorer

#### General
|Operation                   |Keybinding                  |
|----------------------------|----------------------------|
|Switch between panes        |<kbd>Tab</kbd>              |
|Show remotes                |<kbd>g</kbd>                |
|Filter entries within pane  |<kbd>/</kbd>                |
|Sort entries within pane    |<kbd>,</kbd>                |
|Navigate between directories|<kbd>Left/Right</kbd>       |
|Refresh a pane              |<kbd>Ctrl</kbd>+<kbd>r</kbd>|
|Cancel fetching remotes     |<kbd>Ctrl</kbd>+<kbd>x</kbd>|

#### Item selection
|Operation        |Keybinding       |
|-----------------|-----------------|
|Select one item  |<kbd>Space</kbd> |
|Inverse selection|<kbd>a</kbd>     |
|Select all items |<kbd>A</kbd>     |
|Clear selections |<kbd>Escape</kbd>|

#### Operations
|Operation                    |Keybinding  |
|-----------------------------|------------|
|Copy selected items          |<kbd>p</kbd>|
|Move selected items          |<kbd>m</kbd>|
|Delete selected items        |<kbd>d</kbd>|
|Make directory               |<kbd>M</kbd>|
|Generate public link for item|<kbd>;</kbd>|
|Show remote information      |<kbd>i</kbd>|

### Mounts

#### Manager
|Operation  |Keybinding  |
|-----------|------------|
|Create new |<kbd>n</kbd>|
|Unmount    |<kbd>u</kbd>|
|Unmount all|<kbd>U</kbd>|

#### Wizard
|Operation        |Keybinding                  |
|-----------------|----------------------------|
|Create mountpoint|<kbd>Ctrl</kbd>+<kbd>s</kbd>|
|Cancel           |<kbd>Ctrl</kbd>+<kbd>c</kbd>|

### Job Manager
|Operation            |Keybinding                  |
|---------------------|----------------------------|
|Navigate between jobs|<kbd>Down/Up</kbd>          |
|Cancel job           |<kbd>x</kbd>                |
|Cancel job group     |<kbd>Ctrl</kbd>+<kbd>x</kbd>|

## Additional Notes
- To control your local rclone instance, launch `rclone rcd --rc-no-auth`  and use the output host and port to login. Optionally, you can include authentication credentials with `--rc-user` and `--rc-pass` and excluding the `--rc-no-auth` flag.
