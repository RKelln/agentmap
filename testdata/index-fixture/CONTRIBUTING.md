<!-- AGENT:NAV
purpose:contribution guidelines; pull request workflow; dev environment setup
nav[2]{s,n,name,about}:
10,18,##Getting Started,dev environment setup; fork and clone workflow
29,22,##Pull Requests,branch naming conventions; review checklist; merge criteria
-->

# Contributing

We welcome contributions of all kinds: bug fixes; new features; documentation
improvements; and test coverage. Please read this guide before submitting
a pull request.

## Getting Started

Fork the repository and clone your fork locally:

    git clone https://github.com/<you>/myproject
    cd myproject

Install the development dependencies:

    make dev-setup

Run the test suite to verify a clean baseline:

    make test

All tests should pass before you start making changes. If any test fails
on a fresh clone please file an issue before proceeding.

## Pull Requests

Before opening a pull request make sure the following are true:

- All existing tests pass: `make test`
- New code has corresponding tests with table-driven cases
- `make lint` reports no issues
- Commit messages follow Conventional Commits format

Branch naming convention:

    <type>/<short-description>

Examples: `feat/add-index-command`, `fix/parser-code-fence`, `docs/nav-guide`.

Squash fixup commits before requesting review. The PR description should
explain the motivation for the change and link to any relevant issues.
Reviews are typically completed within two business days.
