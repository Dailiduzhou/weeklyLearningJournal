// Simple WebSocket test
console.log('Starting test...');

const ws = new WebSocket('ws://localhost:8080/ws?username=TestBot');

ws.onopen = () => {
    console.log('✓ Connected to server');
    
    // Wait a bit then send a message
    setTimeout(() => {
        console.log('Sending test message...');
        const msg = { type: 'chat', content: 'Hello from bot!' };
        ws.send(JSON.stringify(msg));
        console.log('✓ Message sent');
    }, 1000);
};

ws.onmessage = (event) => {
    console.log('✓ Received:', event.data);
    const msg = JSON.parse(event.data);
    console.log('  Type:', msg.type);
    console.log('  Username:', msg.username);
    console.log('  Content:', msg.content);
    if (msg.usernames) {
        console.log('  Users:', msg.usernames);
    }
};

ws.onerror = (error) => {
    console.error('✗ Error:', error);
};

ws.onclose = () => {
    console.log('✗ Connection closed');
};
