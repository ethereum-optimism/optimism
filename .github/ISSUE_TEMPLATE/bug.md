name: Bug Report
description: Create a bug report
labels: ["C-bug", "S-needs-triage"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report! Please provide as much detail as possible.

  - type: textarea
    id: what-happened
    attributes:
      label: Describe the bug
      description: |
        A clear and concise description of what the bug is.

    validations:
      required: true
  - type: textarea
    id: reproduction-steps
    attributes:
      label: Steps to reproduce
      description: Please provide any steps you think might be relevant to reproduce the bug.
      placeholder: |
        Steps to reproduce:

        1. Start '...'
        2. Then '...'
        3. Check '...'
        4. See error
    validations:
      required: true
  - type: textarea
    id: logs
    attributes:
      label: Logs
      description: |
        If applicable, please provide the logs leading up to the bug.

        **Please also provide debug logs.** By default, these can be found in:

        - `~/.cache/rollup-boost/logs` on Linux
        - `~/Library/Caches/rollup-boost/logs` on macOS
        - `%localAppData%/rollup-boost/logs` on Windows
      render: text
    validations:
      required: false
  - type: dropdown
    id: platform
    attributes:
      label: Platform(s)
      description: What platform(s) did this occur on?
      multiple: true
      options:
        - Linux (x86)
        - Linux (ARM)
        - Mac (Intel)
        - Mac (Apple Silicon)
        - Windows (x86)
        - Windows (ARM)
