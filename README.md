<p align="center">
  <h1 align="center">punchout (Custom Fork)</h1>
  <p align="center">
    <a href="https://github.com/dhth/punchout/actions/workflows/main.yml"><img alt="Build Status" src="https://img.shields.io/github/actions/workflow/status/dhth/punchout/main.yml?style=flat-square"></a>
    <a href="https://github.com/dhth/punchout/actions/workflows/vulncheck.yml"><img alt="Vulnerability Check" src="https://img.shields.io/github/actions/workflow/status/dhth/punchout/vulncheck.yml?style=flat-square&label=vulncheck"></a>
  </p>
</p>

`punchout` takes the suck out of logging time on JIRA. This custom fork includes automated workflow transitions.

<p align="center">
  <img src="https://tools.dhruvs.space/images/punchout/punchout.gif" alt="Usage" />
</p>

💾 Installation
---

### Prerequisites
* **Go**: Ensure you have Go installed on your system (version 1.21 or higher).

### Installing from source
Since this is a custom version with automated Jira transitions, install it directly from this local directory:

1.  Open your terminal in the project root directory.
2.  Run the following command:
    ```sh
    go install .
    ```
3.  Ensure your Go binary directory is in your executable path. Add this to your `~/.bashrc` or `~/.zshrc`:
    ```sh
    export PATH=$PATH:$(go env GOPATH)/bin
    ```

**Note for previous Homebrew users:**
If you previously installed `punchout` via Brew, uninstall it first to avoid path conflicts:
```sh
brew uninstall punchout
hash -r
