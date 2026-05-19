# Contributing to Kiro

Thank you for your interest in contributing to Kiro! This guide will help you get started.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- Node.js 18+ and npm 9+
- Git
- A code editor (we recommend VS Code)

### Setting Up Your Development Environment

1. **Fork the repository** on GitHub

2. **Clone your fork:**
bash
   git clone https://github.com/YOUR_USERNAME/kiro.git
   cd kiro


3. **Add the upstream remote:**
bash
   git remote add upstream https://github.com/kiro-ai/kiro.git


4. **Install dependencies:**
bash
   npm install


5. **Create a branch for your work:**
bash
   git checkout -b feature/your-feature-name


## Development Workflow

### Running Tests

bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage


### Linting and Formatting

bash
# Check for linting issues
npm run lint

# Auto-fix linting issues
npm run lint:fix

# Format code
npm run format


### Building

bash
# Build the project
npm run build

# Build in watch mode
npm run build:watch


## Making Changes

### Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages:

- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `style:` Code style changes (formatting, missing semicolons, etc.)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

**Examples:**
bash
git commit -m "feat: add support for TypeScript 5.0"
git commit -m "fix: resolve memory leak in file watcher"
git commit -m "docs: update installation instructions"


### Pull Request Process

1. **Ensure your code passes all checks:**
bash
   npm run lint
   npm test
   npm run build


2. **Update documentation** if you've changed APIs or added features

3. **Push your branch:**
bash
   git push origin feature/your-feature-name


4. **Create a Pull Request** on GitHub:
   - Use a clear, descriptive title
   - Reference any related issues (e.g., "Fixes #123")
   - Describe what changed and why
   - Include screenshots for UI changes

5. **Respond to review feedback** promptly and professionally

### Keeping Your Fork Updated

bash
# Fetch upstream changes
git fetch upstream

# Merge upstream changes into your main branch
git checkout main
git merge upstream/main

# Push updates to your fork
git push origin main


## Code Style

- Follow the existing code style in the project
- Use TypeScript for type safety
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused

## Testing Guidelines

- Write tests for new features and bug fixes
- Aim for high test coverage
- Use descriptive test names that explain what is being tested
- Follow the Arrange-Act-Assert pattern

## Documentation

- Update README.md for user-facing changes
- Add JSDoc comments for public APIs
- Update CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/) format

## Reporting Issues

When reporting bugs, please include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Your environment (OS, Node version, etc.)
- Relevant logs or error messages

## Feature Requests

We welcome feature requests! Please:

- Check if the feature has already been requested
- Clearly describe the problem you're trying to solve
- Explain how the feature would benefit users
- Consider providing a proposed implementation approach

## Questions?

If you have questions about contributing, feel free to:

- Open a discussion on GitHub
- Reach out to the maintainers
- Check existing issues and pull requests

## License

By contributing to Kiro, you agree that your contributions will be licensed under the same license as the project.

Thank you for contributing to Kiro! 🚀