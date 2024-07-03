/*
 * Performance tests for the Create Short URL endpoint
 *
 * This test file combines load, stress, and spike tests for the Create Short URL endpoint.
 * The test type is determined by the TEST_TYPE environment variable.
 * 
 * Test types:
 * 1. LOAD: Simulates a gradual increase in concurrent users
 * 2. STRESS: Pushes the system beyond its normal capacity
 * 3. SPIKE: Simulates sudden bursts of traffic
 * 
 * Usage: Set the TEST_TYPE environment variable when running the test
 * e.g., k6 run -e TEST_TYPE=LOAD -e BASE_URL=http://localhost:3000 test_create_short_url.js
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

const TEST_TYPE = __ENV.TEST_TYPE || 'LOAD';

const testConfigurations = {
  LOAD: {
    stages: [
      { duration: '30s', target: 20 },
      { duration: '1m', target: 50 },
      { duration: '30s', target: 0 },
    ],
  },
  STRESS: {
    stages: [
      { duration: '30s', target: 50 },
      { duration: '3m', target: 200 },
      { duration: '5m', target: 200 },
      { duration: '30s', target: 0 },
    ],
  },
  SPIKE: {
    stages: [
      { duration: '30s', target: 10 },
      { duration: '1m', target: 100 },
      { duration: '2m', target: 10 },
    ],
  },
};

export const options = {
  stages: testConfigurations[TEST_TYPE].stages,
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.01'], // Error rate should be below 1%
    http_reqs: ['rate>100'], // Throughput should be at least 100 RPS
  },
};

const rtTrend = new Trend('response_time');
const errorRate = new Rate('errors');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

export function setup() {
  console.log(`Starting ${TEST_TYPE} test with BASE_URL: ${BASE_URL}`);
  console.log(`Test configuration: ${JSON.stringify(testConfigurations[TEST_TYPE])}`);
}

let counter = 0;

export default function () {
  const url = `${BASE_URL}/api/v1/short`;
  const payload = JSON.stringify({
    url: `https://www.example.com/${counter++}`,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(url, payload, params);

  rtTrend.add(res.timings.duration);

  const success = check(res, {
    'status is 201': (r) => r.status === 201,
    'response body contains short_url': (r) => r.json('short_url') !== undefined,
  });

  if (success) {
    successfulRequests.add(1);
  } else {
    failedRequests.add(1);
    console.error(`Request failed: ${res.status} ${res.body}, URL: ${url}, Payload: ${payload}`);
  }

  errorRate.add(!success);

  sleep(1);
}

export function handleSummary(data) {
  console.log('Test completed');
  console.log(`Test type: ${TEST_TYPE}`);
  console.log(`Successful requests: ${(data.metrics.successful_requests && data.metrics.successful_requests.values && data.metrics.successful_requests.values.count) || 0}`);
  console.log(`Failed requests: ${(data.metrics.failed_requests && data.metrics.failed_requests.values && data.metrics.failed_requests.values.count) || 0}`);
  console.log(`Average response time: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms`);
  console.log(`95th percentile response time: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
  console.log(`Median response time: ${data.metrics.http_req_duration.values.med.toFixed(2)}ms`);
  console.log(`Max response time: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms`);
  console.log(`Requests per second: ${(data.metrics.http_reqs.values.rate / (data.state.testRunDurationMs / 1000)).toFixed(2)}`);
  console.log(`Error rate: ${data.metrics.errors.values.rate.toFixed(4)}`);

  return {};
}
