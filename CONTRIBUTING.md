# Contributing to GoFileSystem

Thank you for considering contributing to GoFileSystem! This document outlines the guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct, which is to be respectful and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

- **Before submitting a bug report**, please check if the issue has already been reported.
- Fill in the bug report template with all required information.
- Include clear steps to reproduce the issue.
- Provide any relevant logs or error messages.

### Suggesting Enhancements

- **Before suggesting an enhancement**, check if it's already been suggested.
- Provide a clear description of the enhancement.
- Explain why this enhancement would be useful to most users.
- Provide examples of how the enhancement would work.

### Pull Requests

- Pull requests should address an existing issue or a clearly beneficial enhancement.
- Ensure your code follows the project's style guidelines.
- Write tests for any new functionality.
- Make sure all tests pass before submitting your pull request.
- Update documentation as needed.

## Development Workflow

1. Fork the repository
2. Create a new branch for your feature/bugfix: `git checkout -b feature/my-feature` or `git checkout -b fix/my-bugfix`
3. Make your changes
4. Write tests for your changes
5. Run the tests: `go test ./...`
6. Update documentation if necessary
7. Commit your changes with a descriptive commit message
8. Push your branch to your fork
9. Submit a pull request

## Style Guidelines

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions small and focused on a single responsibility

## Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

## Testing

- Write unit tests for all new functionality
- Ensure tests are fast and don't depend on external services
- Mock external dependencies when necessary
- Test edge cases and error conditions

## Documentation

- Update the README.md with any necessary changes
- Add or update code comments
- Document public APIs with proper Go doc comments
- Create examples for complex functionality

## Questions?

If you have any questions about contributing, feel free to open an issue for clarification.

Thank you for contributing to GoFileSystem!
