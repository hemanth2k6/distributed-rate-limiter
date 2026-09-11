import http from 'k6/http';
import { check, sleep } from 'k6';

// Configuration: 50 concurrent virtual users for 30 seconds
export const options = {
  vus: 50,
  duration: '30s',
};

export default function () {
  // To simulate 50 distinct IPs, we use the unique Virtual User ID (__VU)
  // injected by k6 into the X-Forwarded-For header. Our Go middleware 
  // parses this header, creating a unique token bucket for each VU.
  const spoofedIP = `192.168.1.${__VU}`; 
  
  const params = {
    headers: {
      'X-Forwarded-For': spoofedIP,
    },
  };

  const res = http.get('http://localhost:8080/data', params);

  // Track the distribution of 200 OKs vs 429 Too Many Requests
  check(res, {
    'status is 200 (Allowed)': (r) => r.status === 200,
    'status is 429 (Rate Limited)': (r) => r.status === 429,
  });

  // Short sleep to prevent overloading the local OS socket limits
  sleep(0.1);
}
