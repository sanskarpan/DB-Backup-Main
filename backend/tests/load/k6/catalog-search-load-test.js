import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '1m', target: 50 },
    { duration: '5m', target: 200 },
    { duration: '10m', target: 200 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  const searchTerms = ['postgres', 'mysql', 'full', 'incremental', 'prod', 'dev'];
  const filters = ['?status=completed', '?type=full', '?database=postgres', ''];

  const term = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  const filter = filters[Math.floor(Math.random() * filters.length)];

  const res = http.get(`${BASE_URL}/api/v1/backups/search${filter}&q=${term}`, {
    headers: { 'Authorization': `Bearer ${API_KEY}` },
  });

  errorRate.add(res.status !== 200);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has results': (r) => r.json('results') !== undefined,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(0.5);
}
