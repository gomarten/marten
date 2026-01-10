# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in Marten, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email security concerns to the maintainers
3. Include a detailed description of the vulnerability
4. Provide steps to reproduce if possible

We will respond within 48 hours and work with you to understand and address the issue.

## Security Best Practices

When using Marten:

- Always use HTTPS in production
- Use the `Secure` middleware for security headers
- Use the `RateLimit` middleware to prevent abuse
- Validate and sanitize all user input
- Use `BodyLimit` middleware to prevent large payload attacks
- Keep Marten and Go updated to the latest versions
