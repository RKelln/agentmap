# My Project

A brief description of what this project does and why it exists.

## Installation

To install, clone the repository and run the build tool:

    git clone https://github.com/example/myproject
    cd myproject
    make build

The binary will be placed in the project root. No external dependencies are
required at runtime. The build requires Go 1.21 or later.

Verify the installation by running the version command:

    ./myproject version

If the command prints a version string the installation succeeded. If you see
an error check that your PATH includes the Go binary directory.

## Usage

Basic usage requires specifying a subcommand and a target path:

    ./myproject <command> [flags] [path]

Available subcommands are documented in the sections below. All commands
accept a `--dry-run` flag which previews changes without writing any files.

Common flags:

- `--config` — path to a custom agentmap.yml configuration file
- `--dry-run` — show what would be written without modifying files
- `--quiet` — suppress informational output

## Configuration

Configuration is loaded from `agentmap.yml` in the project root. If no
config file is found the tool uses built-in defaults. All fields are
optional.

Example configuration:

    min_lines: 50
    max_depth: 3
    exclude:
      - CHANGELOG.md
      - vendor/**

The `exclude` field accepts glob patterns. Directories are matched by
their path prefix. The `.agentmap/` directory is always excluded
regardless of the `exclude` setting.

See the design document for a complete reference of all configuration
fields and their default values.
