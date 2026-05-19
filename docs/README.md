# Kiro

Kiro is an AI-powered development environment that writes code so you can focus on designing systems, exploring solutions, and making decisions.

## Features

- **Direct Code Editing**: Edit files and code directly in your project
- **Terminal Integration**: Run commands and automate CLI tasks
- **Multi-Language Support**: Work with any programming language or framework
- **Context-Aware**: Understands your project structure and conventions
- **Safety-First**: Confirms high-risk operations before execution
- **Verification Built-In**: Automatically builds and tests changes

## Getting Started

### Installation

bash
npm install -g kiro


### Quick Start

1. Navigate to your project directory:
bash
   cd your-project


2. Start Kiro:
bash
   kiro


3. Start coding with natural language:

   "Add error handling to the login function"
   "Refactor the user service to use async/await"
   "Write tests for the payment module"


## How It Works

Kiro operates as your development partner:

1. **Understand**: Reads your codebase and understands context
2. **Plan**: Determines the best approach for your request
3. **Implement**: Makes changes directly to your files
4. **Verify**: Runs builds and tests to ensure correctness

## Best Practices

### Be Specific

✅ "Add input validation to the createUser function in src/auth.ts"
❌ "Make the code better"


### Provide Context

✅ "Refactor the API client to use axios instead of fetch, matching the pattern in src/services/api.ts"
❌ "Change the HTTP library"


### Review Changes
Kiro shows you what changed. Review the diffs before committing.

## Safety Features

- **Reversible by Default**: Local file edits are easily undone
- **Confirmation for High-Risk**: Asks before destructive operations
- **Git-Safe**: Never force-pushes or modifies history without permission
- **Secret Protection**: Avoids echoing credentials or tokens

## Configuration

Create a `.kirorc.json` in your project root:


{
  "autoVerify": true,
  "testCommand": "npm test",
  "buildCommand": "npm run build",
  "excludePaths": ["node_modules", "dist", ".git"]
}


## Examples

### Fix a Bug

"The login function in src/auth.ts throws an error when email is undefined. Add validation."


### Add a Feature

"Add pagination to the /api/users endpoint. Use limit/offset params and return metadata."


### Refactor Code

"Convert all class components in src/components to functional components with hooks."


### Write Tests

"Write unit tests for the UserService class. Cover all public methods."


## Troubleshooting

### Kiro isn't finding my files
- Ensure you're in the project root directory
- Check `.kirorc.json` for overly broad `excludePaths`

### Changes aren't being verified
- Verify `testCommand` and `buildCommand` in `.kirorc.json`
- Ensure dependencies are installed

### Kiro is too cautious
- For trusted operations, explicitly state "proceed without confirmation"
- Adjust safety settings in configuration

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Documentation: [https://kiro.dev/docs](https://kiro.dev/docs)
- Issues: [https://github.com/kiro/kiro/issues](https://github.com/kiro/kiro/issues)
- Community: [https://discord.gg/kiro](https://discord.gg/kiro)

---

Built with ❤️ by the Kiro team