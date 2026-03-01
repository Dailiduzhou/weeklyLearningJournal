# SSE Server

A Go-based Server-Sent Events (SSE) server with JWT authentication, CORS support, and a complete frontend demo.

## Features

- Real-time score board updates via SSE
- JWT-based authentication
- Configurable CORS origins
- Message history for reconnection support
- Automatic heartbeat for connection keep-alive
- Interactive frontend demo with event-source-polyfill
- Cross-browser compatibility

## Quick Start

```bash
# Start both backend and frontend
./start.sh

# Or start manually
go run main.go &
cd frontend && python3 -m http.server 3000
```

Then open http://localhost:3000 in your browser.

## Project Structure

```
sse/
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── .env.example         # Environment variables template
├── config/
│   └── config.go        # Configuration management
├── controller/
│   └── controller.go    # HTTP handlers
├── middleware/
│   └── middleware.go    # CORS and auth middleware
├── model/
│   └── model.go         # Data models and broker
├── utils/
│   └── utils.go         # JWT utilities
└── frontend/
    ├── package.json     # Frontend dependencies
    ├── index.html       # SSE client demo
    └── node_modules/    # npm dependencies
```

## Setup

1. Install Go 1.21 or higher
2. Install dependencies:
   ```bash
   go mod download
   ```

3. Configure environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your values
   ```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `:8080` |
| `JWT_SECRET` | JWT signing secret | `18e9ba09a3566f97de630c21331ebc11` |
| `CORS_ORIGIN` | Allowed CORS origin | `http://localhost:3000` |
| `TOKEN_ISSUER` | JWT token issuer | `dailiduzhou` |
| `TOKEN_SUBJECT` | JWT token subject | `Billboard` |

## Running the Server

### Quick Start (Backend + Frontend)

Use the provided start script to run both servers:

```bash
./start.sh
```

This will:
- Start the backend server on `http://localhost:8080`
- Start the frontend demo on `http://localhost:3000`
- Open http://localhost:3000 in your browser

### Backend Only

```bash
go run main.go
```

Server will start on `http://localhost:8080`

## API Endpoints

### GET /events

Subscribe to real-time score board updates.

**Query Parameters:**
- `token` (required): JWT authentication token

**Headers:**
- `Authorization`: `Bearer <token>` (alternative to query param)
- `Last-Event-ID`: Resume from last received message ID

## Authentication

Generate a JWT token using the utils package:

```go
import "sse/utils"

token, err := utils.GenerateToken(userID)
```

The token expires after 2 hours.

## Example Client

```javascript
const eventSource = new EventSource('http://localhost:8080/events?token=YOUR_JWT_TOKEN');

eventSource.onmessage = (event) => {
    console.log('Message:', event.data);
};

eventSource.onerror = (error) => {
    console.error('SSE error:', error);
};
```

## Frontend Demo

A complete frontend demo is included using `event-source-polyfill` for cross-browser compatibility.

### Setup Frontend

```bash
cd frontend
npm install
```

### Running the Demo

**Option 1: Using the start script**
```bash
./start.sh
```

**Option 2: Manual setup**
Open `frontend/index.html` in a web browser:

```bash
# Serve with a local server (e.g., using Python)
cd frontend
python3 -m http.server 3000
```

Then open http://localhost:3000 in your browser.

### Demo Features

- **JWT Token Generation**: Generate tokens directly in the browser
- **Real-time Connection**: Connect to SSE endpoint with authentication
- **Message Display**: View all received messages with timestamps
- **Connection Stats**: Track message count, last event ID, and connection time
- **Auto-reconnection**: Resumes from last event ID on reconnection

### Using event-source-polyfill

For better browser compatibility, especially in older browsers:

```javascript
import { EventSourcePolyfill } from 'event-source-polyfill';

const eventSource = new EventSourcePolyfill(
    'http://localhost:8080/events?token=YOUR_JWT_TOKEN',
    {
        heartbeatTimeout: 30000,
        withCredentials: true
    }
);

eventSource.onmessage = (event) => {
    console.log('Message:', event.data);
    console.log('Last Event ID:', event.lastEventId);
};
```

## Building

```bash
go build -o sse-server
```

## Development

Run tests:
```bash
go test ./...
```

Build with optimizations:
```bash
go build -ldflags="-s -w" -o sse-server
```
