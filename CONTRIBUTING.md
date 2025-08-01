# Contributing to B2B Trust Broker

Thank you for your interest in contributing to the B2B Trust Broker platform! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Node.js >= 18.0
- Docker >= 20.0
- Kubernetes cluster (minikube or cloud)
- AWS CLI configured
- Terraform >= 1.0

### Local Development Setup

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd pn-test
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Set up local environment:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Start local development:
   ```bash
   npm run dev
   ```

## Development Workflow

### Branch Naming Convention

- `feature/component-name` - New features
- `bugfix/issue-description` - Bug fixes
- `hotfix/critical-fix` - Critical fixes
- `docs/documentation-update` - Documentation updates

### Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes
- `refactor` - Code refactoring
- `test` - Test changes
- `chore` - Build/tooling changes

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes following the code standards
3. Write/update tests for your changes
4. Update documentation as needed
5. Submit a pull request with a clear description
6. Ensure all CI checks pass
7. Get code review approval
8. Merge after approval

## Code Standards

### General Principles

- Write clean, readable, and maintainable code
- Follow SOLID principles
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions small and focused

### Language-Specific Standards

#### TypeScript/JavaScript
- Use TypeScript for all new code
- Follow ESLint configuration
- Use async/await instead of promises
- Prefer const over let, avoid var

#### Python
- Follow PEP 8 style guide
- Use type hints
- Write docstrings for functions
- Use virtual environments

#### Go
- Follow Go formatting (gofmt)
- Use meaningful package names
- Write tests for exported functions
- Use context for cancellation

### Testing Standards

- Write unit tests for all new functionality
- Aim for >80% code coverage
- Use descriptive test names
- Mock external dependencies
- Test both success and error cases

## Testing

### Running Tests

```bash
# Run all tests
npm test

# Run tests with coverage
npm run test:coverage

# Run specific test suite
npm run test:unit
npm run test:integration
npm run test:e2e
```

### Test Structure

- `tests/unit/` - Unit tests
- `tests/integration/` - Integration tests
- `tests/e2e/` - End-to-end tests
- `tests/fixtures/` - Test data and fixtures

## Documentation

### Documentation Standards

- Keep documentation up to date
- Use clear and concise language
- Include code examples where appropriate
- Update README files for significant changes
- Document API changes in CHANGELOG.md

### Required Documentation

- README.md for each component
- API documentation
- Architecture diagrams
- Deployment guides
- Troubleshooting guides

## Review Process

### Code Review Checklist

- [ ] Code follows project standards
- [ ] Tests are included and passing
- [ ] Documentation is updated
- [ ] No security vulnerabilities
- [ ] Performance impact considered
- [ ] Error handling is appropriate

### Review Guidelines

- Be constructive and respectful
- Focus on the code, not the person
- Suggest improvements clearly
- Approve only when satisfied
- Request changes when needed

## Getting Help

- Check existing documentation
- Search existing issues
- Ask questions in discussions
- Contact maintainers for urgent issues

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project. 