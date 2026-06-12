import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const backupDuration = new Trend('backup_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },  // Ramp up to 10 users
    { duration: '5m', target: 50 },  // Ramp up to 50 users
    { duration: '10m', target: 50 }, // Stay at 50 users
    { duration: '3m', target: 100 }, // Spike to 100 users
    { duration: '5m', target: 50 },  // Scale down to 50
    { duration: '2m', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'],  // 95% of requests must complete below 5s
    http_req_failed: ['rate<0.05'],     // Error rate must be below 5%
    errors: ['rate<0.1'],               // Custom error rate
    backup_duration: ['p(99)<30000'],   // 99% of backups complete within 30s
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export function setup() {
  // Setup phase - create test database if needed
  const setupRes = http.post(`${BASE_URL}/api/v1/databases`, JSON.stringify({
    name: `load-test-db-${Date.now()}`,
    type: 'postgres',
    host: 'postgres',
    port: 5432,
    username: 'testuser',
    password: 'testpass',
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${API_KEY}`,
    },
  });

  return { databaseId: setupRes.json('id') };
}

export default function (data) {
  const databaseTypes = ['postgres', 'mysql', 'mongodb', 'redis'];
  const backupTypes = ['full', 'incremental', 'differential'];

  // Select random database and backup type
  const dbType = databaseTypes[Math.floor(Math.random() * databaseTypes.length)];
  const backupType = backupTypes[Math.floor(Math.random() * backupTypes.length)];

  const payload = JSON.stringify({
    database: `test-db-${dbType}-${__VU}`,
    type: backupType,
    compress: true,
    encrypt: true,
    tags: [`load-test`, `vu-${__VU}`, `iter-${__ITER}`],
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${API_KEY}`,
    },
    timeout: '60s',
  };

  // Execute backup request
  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/backups`, payload, params);
  const duration = Date.now() - startTime;

  // Record metrics
  backupDuration.add(duration);
  errorRate.add(res.status !== 200 && res.status !== 201);

  // Assertions
  const checkResult = check(res, {
    'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    'response has backup_id': (r) => r.json('backup_id') !== undefined,
    'response time < 10s': (r) => r.timings.duration < 10000,
  });

  if (!checkResult) {
    console.error(`Backup request failed: ${res.status} ${res.body}`);
  }

  // Poll backup status
  if (res.status === 200 || res.status === 201) {
    const backupId = res.json('backup_id');

    for (let i = 0; i < 10; i++) {
      const statusRes = http.get(`${BASE_URL}/api/v1/backups/${backupId}`, {
        headers: {
          'Authorization': `Bearer ${API_KEY}`,
        },
      });

      const status = statusRes.json('status');

      if (status === 'completed' || status === 'failed') {
        check(statusRes, {
          'backup completed successfully': (r) => r.json('status') === 'completed',
        });
        break;
      }

      sleep(2);
    }
  }

  // Think time between iterations
  sleep(Math.random() * 3 + 1); // 1-4 seconds
}

export function teardown(data) {
  // Cleanup phase
  console.log('Load test completed');
}
