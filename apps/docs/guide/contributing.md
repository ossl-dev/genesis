# Contributing

Thank you for your interest in contributing to Genesis!

## Ways to Contribute

- 🐛 Report bugs
- 💡 Suggest features
- 📝 Improve documentation
- 🔧 Fix issues
- 🧩 Create plugins
- ⭐ Star the project

## Getting Started

### 1. Fork the Repository

Fork [Genesis on GitHub](https://github.com/Open-Vanguard/genesis)

### 2. Clone Your Fork

```bash
git clone https://github.com/YOUR_USERNAME/genesis.git
cd genesis
```

### 3. Install Dependencies

```bash
bun install
```

### 4. Build Packages

```bash
bun run build
```

### 5. Make Your Changes

Create a new branch:

```bash
git checkout -b feature/my-feature
```

### 6. Test Your Changes

```bash
# Build packages
bun run build

# Test locally
cd examples
bunx genesis apply
```

### 7. Submit a Pull Request

```bash
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature
```

Then open a PR on GitHub!

## Development Setup

See the [Development Setup Guide](/guide/development-setup) for detailed instructions.

## Code Style

### TypeScript

- Use TypeScript for all code
- Follow existing code style
- Add types for all functions
- Use ESM imports

### Formatting

```bash
# Format code (if configured)
bun run format
```

### Linting

```bash
# Lint code
bun run lint
```

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new plugin
fix: resolve installation issue
docs: update README
chore: update dependencies
```

## Pull Request Guidelines

### Before Submitting

- ✅ Code builds successfully
- ✅ Tests pass (if applicable)
- ✅ Documentation updated
- ✅ Commit messages follow convention
- ✅ PR description explains changes

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Breaking change

## Testing
How did you test this?

## Checklist
- [ ] Code builds
- [ ] Documentation updated
- [ ] Follows code style
```

## Creating Plugins

Want to create a plugin? See:

- [Plugin Development Guide](/guide/plugin-development)
- [Creating a Plugin](/plugins/creating-plugin)
- [Best Practices](/plugins/best-practices)

## Reporting Bugs

### Before Reporting

1. Search [existing issues](https://github.com/Open-Vanguard/genesis/issues)
2. Check [troubleshooting guide](/guide/troubleshooting)
3. Try latest version

### Bug Report Template

```markdown
## Description
Clear description of the bug

## To Reproduce
Steps to reproduce:
1. Create config with...
2. Run genesis apply
3. See error

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: macOS 14.0
- Genesis version: 0.1.0
- Node version: 20.10.0

## Configuration
```typescript
// Your genesis.config.ts
```

## Error Output
```
// Full error message
```
```

## Suggesting Features

### Feature Request Template

```markdown
## Feature Description
Clear description of the feature

## Use Case
Why is this feature needed?

## Proposed Solution
How should it work?

## Alternatives Considered
Other approaches you've thought about
```

## Documentation

### Improving Docs

Documentation is in `apps/docs/`:

```bash
# Run docs locally
bun run docs:dev

# Build docs
bun run docs:build
```

### Documentation Style

- Use clear, concise language
- Include code examples
- Add links to related pages
- Use proper markdown formatting

## Code of Conduct

Be respectful and inclusive. We're all here to learn and build together.

## Questions?

- [GitHub Discussions](https://github.com/Open-Vanguard/genesis/discussions)
- [GitHub Issues](https://github.com/Open-Vanguard/genesis/issues)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Thank You!

Every contribution helps make Genesis better. Thank you for your time and effort! 🙏

