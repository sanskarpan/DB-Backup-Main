import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const restoreDuration = new Trend('restore_duration');

export const options = {
  stages: [
    { duration: '1m', target: 5 },
    { duration: '3m', target: 20 },
    { duration: '5m', target: 20 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<10000'],
    http_req_failed: ['rate<0.05'],
    restoreDuration: ['p(99)<60000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  const payload = JSON.stringify({
    backup_id: `backup-${Math.floor(Math.random() * 100)}`,
    target_database: `restore-target-${__VU}`,
    point_in_time: new Date(Date.now() - Math.random() * 86400000).toISOString(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${API_KEY}`,
    },
  };

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/restore`, payload, params);
  const duration = Date.now() - startTime;

  restoreDuration.add(duration);
  errorRate.add(res.status !== 200);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has restore_id': (r) => r.json('restore_id') !== undefined,
  });

  sleep(Math.random() * 5 + 2);
}
