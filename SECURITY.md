# Security Policy

## Supported Versions

Limoni is pre-1.0. Security fixes land on `main` and ship in the next release of
the latest minor line; older minor lines are not patched.

| Version | Supported          |
| ------- | ------------------ |
| latest `v0.x` minor (currently `v0.3.x`) | :white_check_mark: |
| older `v0.x` minors | :x: |

Once `v1.0.0` is tagged, the latest `v1.x` minor will be supported the same way.

---

## Reporting a Vulnerability

We take the security of Limoni very seriously. If you believe you have found a security vulnerability in Limoni, please report it to us as described below.

### 🔒 How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report security issues through one of the following channels:
1. **GitHub Security Advisory**: Open a private advisory on the [Limoni Security Tab](https://github.com/thebanri/limoni/security/advisories/new).
2. **Email**: Send an email directly to [thebanri@gmail.com](mailto:thebanri@gmail.com) with the subject `[SECURITY] Limoni Vulnerability Report`.

Please include the following details in your report:
- Type of issue (e.g. buffer overrun, terminal escape sequence injection, resource exhaustion / DoS, unsafe concurrency).
- Steps to reproduce the issue (proof of concept code or terminal input trace).
- Affected Limoni versions and target OS / terminal emulator.
- Any potential mitigations or patch suggestions.

---

## Response Timeline

- **Initial Response:** Within 48 hours to acknowledge receipt of the report.
- **Triage & Status Update:** Within 5 business days with an assessment and fix timeline.
- **Fix & Public Disclosure:** Coordinated release with credit given to the reporter in the changelog and security advisory (unless requested to remain anonymous).
