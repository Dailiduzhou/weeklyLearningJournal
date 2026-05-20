import http from 'k6/http';
import { check, sleep } from 'k6';

const baseUrl = __ENV.BASE_URL || 'http://product:8000';
const seckillPath = __ENV.SECKILL_PATH || '/api/seckill';
const userCount = Number(__ENV.USER_COUNT || 50000);
const startUserID = Number(__ENV.START_USER_ID || 1);
const sleepMs = Number(__ENV.SLEEP_MS || 0);

function buildStages() {
  if (__ENV.K6_STAGES) {
    return __ENV.K6_STAGES.split(',').map((entry) => {
      const parts = entry.split(':');
      if (parts.length !== 2 || !parts[0] || Number.isNaN(Number(parts[1]))) {
        throw new Error(`invalid K6_STAGES entry: "${entry}", expected format like 30s:200`);
      }
      return { duration: parts[0], target: Number(parts[1]) };
    });
  }

  return [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ];
}

export const options = {
  stages: buildStages(),
  thresholds: {
    http_req_failed: ['rate<0.20'],
    http_req_duration: ['p(95)<1000'],
  },
};

function nextUserID() {
  return ((__VU * 1000000 + __ITER) % userCount) + startUserID;
}

export default function () {
  const payload = JSON.stringify({ userID: nextUserID() });
  const res = http.post(`${baseUrl}${seckillPath}`, payload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: __ENV.HTTP_TIMEOUT || '5s',
  });

  check(res, {
    'status < 500': (r) => r.status < 500,
    'has body': (r) => r.body && r.body.length > 0,
  });

  if (sleepMs > 0) {
    sleep(sleepMs / 1000);
  }
}
