# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Frontend demo with `event-source-polyfill` integration
- Simple test page (`test.html`) for quick verification
- JWT token generation using browser's Web Crypto API
- CDN integration for event-source-polyfill (no build tools required)
- Reconnection support with `Last-Event-ID` header
- Message history handling (up to 50 messages) on backend
- Comprehensive documentation (README.md, QUICKSTART.md, frontend/README.md)
- Start script (`start.sh`) to run both servers

### Fixed
- **Backend**:
  - Fixed `ServerHTTP` → `ServeHTTP` (http.Handler interface)
  - Fixed CORS origin: `http:localhost:3000` → `http://localhost:3000`
  - Fixed CORS header typos: `Access-Contreol-*` → `Access-Control-*`
  - Fixed log typos: `Hearbeat` → `Heartbeat`
  - Fixed redundant `.String()` on time format
  - Fixed Go version: `1.25.6` → `1.21`
  - Added message history sending on reconnection

- **Frontend**:
  - Removed ES6 import (browser incompatibility)
  - Added CDN version of event-source-polyfill
  - Added proper error handling and display
  - Added event listeners instead of inline onclick
  - Added JWT generation using Web Crypto API
  - Improved connection status handling
  - Added visual feedback for errors

### Changed
- **Project Structure**:
  - Moved `NewBroker()` from utils to model package
  - Created config package for centralized configuration
  - Added frontend directory with demo pages

- **Configuration**:
  - Made CORS origin configurable via environment variable
  - Made port configurable via environment variable
  - Updated .env.example with new variables
  - All configuration now in config package

- **Documentation**:
  - Unified language to English
  - Added comprehensive README.md
  - Added QUICKSTART.md for beginners
  - Added frontend/README.md for frontend-specific info
  - Added this CHANGELOG.md

### Removed
- Chinese logs in utils.go (converted to English)
- Hardcoded configuration values

## [0.1.0] - Initial Release

### Features
- Basic SSE server with Go
- JWT authentication
- CORS middleware
- Message broker with history
- Heartbeat for connection keep-alive
