# Quick Start Guide

## 🚀 Get Started in 3 Minutes

### Prerequisites

- Go 1.21 or higher
- Modern web browser (Chrome, Firefox, Safari, Edge)
- Python 3 (optional, for serving frontend)

### Step 1: Start the Servers

The easiest way is to use the provided start script:

```bash
./start.sh
```

This will automatically:
- Start the Go backend server on port 8080
- Start the Python HTTP server on port 3000
- Display the URLs to access both services

### Step 2: Open the Frontend

Navigate to http://localhost:3000/index.html in your browser.

**Alternative**: Use the quick test page at http://localhost:3000/test.html

### Step 3: Generate a Token and Connect

1. Enter a User ID (default: 12345)
2. Click the "Generate Token" button
3. Click the "Connect" button
4. Watch the real-time score board updates!

## 📊 What You'll See

- **Live Messages**: Score board updates every 2 seconds
- **Connection Stats**: Message count, last event ID, connection time
- **Full Control**: Connect/disconnect buttons, custom user IDs
- **Error Handling**: Clear error messages when something goes wrong

## 🔧 Manual Setup

If you prefer to set up manually:

### Backend

```bash
# Terminal 1
go run main.go
```

### Frontend

```bash
# Terminal 2
cd frontend
python3 -m http.server 3000
```

Then open http://localhost:3000/index.html in your browser.

## 📝 Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
PORT=:8080
JWT_SECRET=your_secret_here
CORS_ORIGIN=http://localhost:3000
TOKEN_ISSUER=dailiduzhou
TOKEN_SUBJECT=Billboard
```

## 🎯 Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check [frontend/README.md](frontend/README.md) for frontend-specific guide
- Explore the code and customize for your use case

## 💡 Tips

- The demo automatically generates JWT tokens using browser's Web Crypto API
- Messages are stored for reconnection (up to 50 messages)
- Use browser DevTools to inspect SSE events in the Network tab (F12)
- Connection automatically resumes from the last event ID
- Tokens are valid for 2 hours

## ✅ Fixed Issues

The frontend has been updated to fix several issues:

1. **CDN Integration**: Uses jsDelivr CDN for event-source-polyfill (no build tools required)
2. **Improved Error Handling**: Clear error messages displayed to users
3. **JWT Generation**: Uses browser's native Web Crypto API
4. **Event Listeners**: Proper event handling with addEventListener
5. **Reconnection Support**: Backend now sends missed messages on reconnect
6. **Test Page**: Added simple test.html for quick verification

## ❓ Troubleshooting

**Port already in use?**
```bash
# Kill any existing processes
pkill -f "go run main.go"
pkill -f "python3 -m http.server"

# Then start again
./start.sh
```

**CORS errors?**
- Ensure `CORS_ORIGIN` in `.env` matches your frontend URL
- Restart backend after changing `.env` file
- Check browser console (F12) for specific error messages

**No messages appearing?**
- Wait at least 2-3 seconds (server sends messages every 2 seconds)
- Verify the backend is running (check terminal logs)
- Check if the token is valid and not expired
- Look at the browser console for errors (F12 → Console)

**Token generation fails?**
- Ensure User ID is a valid number
- Try using a different browser
- Check that browser supports Web Crypto API (most modern browsers do)

**Connection fails immediately?**
- Verify backend is running on port 8080
- Check that JWT_SECRET in `.env` matches the hardcoded one in HTML
- Try using test.html page which has simpler logic

Need more help? Check the full documentation in [README.md](README.md) or [frontend/README.md](frontend/README.md).
