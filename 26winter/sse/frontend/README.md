# Frontend Demo Guide

## Overview

This frontend demo provides a complete interactive example for connecting to the SSE server. It uses `event-source-polyfill` via CDN for maximum browser compatibility without requiring build tools.

## Features

- **JWT Token Generation**: Generate authentication tokens directly in browser using Web Crypto API
- **Real-time Connection**: Connect and disconnect from SSE endpoint with error handling
- **Message Display**: View all received messages with timestamps and event IDs
- **Connection Statistics**: Track message count, last event ID, and connection duration
- **Auto-reconnection Support**: Stores last event ID for seamless reconnection
- **Error Display**: Shows connection errors and warnings

## Files

- `index.html` - Full-featured SSE client demo
- `test.html` - Simple test page for quick verification
- `test-jwt.js` - Node.js script to generate JWT tokens for testing
- `package.json` - npm dependencies (event-source-polyfill for reference)

## How to Use

### 1. Start the Servers

**Option 1 - Using the start script (recommended):**
```bash
./start.sh
```

**Option 2 - Manual start:**
```bash
# Terminal 1 - Backend
go run main.go

# Terminal 2 - Frontend
cd frontend
python3 -m http.server 3000
```

### 2. Open the Demo

Navigate to one of these pages in your browser:

- **Full Demo**: http://localhost:3000/index.html
- **Quick Test**: http://localhost:3000/test.html

### 3. Using the Full Demo (index.html)

#### Generate a JWT Token

1. Enter a User ID (default: 12345)
2. Click "Generate Token" button
3. A token will be generated and displayed in the token field
4. Token is valid for 2 hours

#### Connect to the Server

1. Verify the Server URL is correct (default: http://localhost:8080/events)
2. Click "Connect" button
3. The status will change to "Connected"
4. Messages will start appearing in the messages section (updated every 2 seconds)

#### Monitor Real-time Updates

- **Messages Received**: Total count of messages received
- **Last Event ID**: ID of the most recent message (useful for reconnection)
- **Connection Time**: Duration of current connection

#### Disconnect

Click "Disconnect" button to close the connection.

### 4. Using the Quick Test (test.html)

The test page is simpler and provides:
- Auto-generated JWT tokens
- Real-time connection logging
- Visual status indicators
- Easy-to-read message logs

## Reconnection Feature

The demo supports automatic reconnection with message history:

1. When you connect, the browser stores the last event ID received
2. If you disconnect and reconnect:
   - The browser sends the `Last-Event-ID` header
   - The server sends any messages you missed
   - You resume from where you left off
3. This ensures no messages are lost during brief disconnections

## Troubleshooting

### Connection Fails

**Symptoms**: Status shows "Disconnected" or error message appears

**Solutions**:
- Ensure backend server is running on port 8080
- Check CORS settings in backend's `.env` file (should match your frontend URL)
- Verify JWT token is valid and not expired (tokens expire after 2 hours)
- Check browser console (F12) for detailed error messages

### No Messages Received

**Symptoms**: Status shows "Connected" but no messages appear

**Solutions**:
- The server sends messages every 2 seconds automatically
- Wait at least 2-3 seconds for first message
- Check backend logs for any errors
- Verify the broker is broadcasting messages
- Try refreshing the page

### Token Generation Fails

**Symptoms**: Click "Generate Token" shows error message

**Solutions**:
- Ensure User ID is a valid number
- Check browser console for JavaScript errors (F12)
- Verify browser supports Web Crypto API (most modern browsers do)
- Try using a different browser

### CORS Errors

**Symptoms**: Browser console shows CORS-related errors

**Solutions**:
- Check backend's `.env` file for `CORS_ORIGIN` setting
- Ensure it matches your frontend URL (e.g., `http://localhost:3000`)
- Restart the backend after changing `.env` file

## Browser Compatibility

The demo uses `event-source-polyfill` for maximum compatibility:

- ✅ Chrome/Edge (latest 2 versions)
- ✅ Firefox (latest 2 versions)
- ✅ Safari (latest 2 versions)
- ✅ Opera (latest version)
- ✅ Older browsers with polyfill support

**Required browser features**:
- ES6 JavaScript
- Web Crypto API (for JWT signing)
- Fetch API or EventSource API

## Customization

### Change Server URL

Update the "Server URL" field in the demo page to point to your SSE endpoint.

### Change User ID

Enter a different User ID and click "Generate Token" again.

### Customize Styling

The demo uses inline CSS in `index.html`. You can modify:
- Colors and fonts in the `<style>` section
- Layout and spacing in the HTML structure
- Add your own CSS classes

### Change Token Expiration

Edit the `generateToken()` function in `index.html`:

```javascript
// Change from 2 hours (2 * 60 * 60) to your desired duration
exp: Math.floor(Date.now() / 1000) + (2 * 60 * 60)
```

## API Integration

If you want to integrate SSE into your own application:

### Basic Usage

```javascript
import { EventSourcePolyfill } from 'event-source-polyfill';

const eventSource = new EventSourcePolyfill(
    'http://localhost:8080/events?token=YOUR_JWT_TOKEN',
    {
        heartbeatTimeout: 30000,
        withCredentials: true
    }
);

eventSource.onopen = () => {
    console.log('Connected to SSE server');
};

eventSource.onmessage = (event) => {
    console.log('Message:', event.data);
    console.log('Event ID:', event.lastEventId);
};

eventSource.onerror = (error) => {
    console.error('SSE error:', error);
};
```

### With Reconnection Support

```javascript
let lastEventId = null;

function connectWithReconnect(token) {
    const url = new URL('http://localhost:8080/events');
    url.searchParams.append('token', token);
    
    if (lastEventId) {
        url.searchParams.append('Last-Event-ID', lastEventId);
    }
    
    const eventSource = new EventSourcePolyfill(url.toString());
    
    eventSource.onmessage = (event) => {
        lastEventId = event.lastEventId;
        // Handle message
    };
}
```

## Testing JWT Tokens

You can generate a test JWT token using the provided Node.js script:

```bash
cd frontend
node test-jwt.js
```

This will output a valid JWT token that you can use for testing.

## Security Notes

⚠️ **Important Security Considerations**:

- The demo generates tokens in the **browser** for testing purposes only
- In **production**, tokens should be generated on your backend server
- Never expose your JWT secret key in frontend code
- Use HTTPS in production for secure connections
- Implement proper token validation and refresh mechanisms
- Store tokens securely (e.g., HttpOnly cookies) in production

## Architecture

### How It Works

1. **Frontend**:
   - Generates JWT token using Web Crypto API
   - Creates SSE connection with authentication
   - Stores last event ID for reconnection
   - Displays messages in real-time

2. **Backend**:
   - Validates JWT token on each connection
   - Manages message history (up to 50 messages)
   - Sends missed messages on reconnection
   - Broadcasts messages every 2 seconds

3. **Communication Flow**:
   ```
   Frontend --(JWT token)--> Backend
   Frontend --(Connect)--> Backend
   Backend --(Messages)--> Frontend (every 2s)
   Frontend --(Last-Event-ID)--> Backend (on reconnect)
   Backend --(Missed messages)--> Frontend
   ```

## Advanced Features

### Message History

The backend keeps the last 50 messages. If you disconnect and reconnect:
1. Send the last event ID you received
2. Server sends all messages after that ID
3. You won't miss any messages

### Heartbeat

The server sends a `: keep-alive` message every 15 seconds to:
- Prevent connection timeout
- Detect dead connections
- Maintain active connection

### Custom Events

You can extend the demo to handle different event types:

```javascript
eventSource.addEventListener('custom-event', (event) => {
    console.log('Custom event:', event.data);
});
```

## Getting Help

If you encounter issues:

1. Check the browser console (F12) for JavaScript errors
2. Check the backend terminal for server errors
3. Review the troubleshooting section above
4. Try the simple test page (`test.html`) first
5. Ensure all servers are running on correct ports
