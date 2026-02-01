// Test JWT generation in Node.js
const crypto = require('crypto');

function base64UrlEncode(str) {
    return Buffer.from(str)
        .toString('base64')
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '');
}

function signHMAC(data, secret) {
    return crypto
        .createHmac('sha256', secret)
        .update(data)
        .digest('base64')
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '');
}

function generateToken(userId) {
    const secret = '18e9ba09a3566f97de630c21331ebc11';
    const header = { alg: 'HS256', typ: 'JWT' };
    const payload = {
        UserID: parseInt(userId),
        iss: 'dailiduzhou',
        sub: 'Billboard',
        exp: Math.floor(Date.now() / 1000) + (2 * 60 * 60)
    };

    const encodedHeader = base64UrlEncode(JSON.stringify(header));
    const encodedPayload = base64UrlEncode(JSON.stringify(payload));
    const signature = signHMAC(`${encodedHeader}.${encodedPayload}`, secret);

    return `${encodedHeader}.${encodedPayload}.${signature}`;
}

const token = generateToken('12345');
console.log('Generated JWT Token:');
console.log(token);
console.log('\nUse this token in the frontend demo.');
