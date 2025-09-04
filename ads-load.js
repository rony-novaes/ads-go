import http from 'k6/http';
import { check, sleep } from 'k6';

// --------- CONFIG ---------
export const options = {
  scenarios: {
    redirect_scenario: {
      executor: 'ramping-arrival-rate',
      startRate: 50,                  // req/s inicial
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 1000,
      stages: [
        { target: 200, duration: '2m' },  // aquecimento
        { target: 600, duration: '4m' },  // carga
        { target: 1000, duration: '4m' }, // estresse
        { target: 0, duration: '30s' },   // cooldown
      ],
      exec: 'redirectTest',
    },
    json_scenario: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 1000,
      stages: [
        { target: 200, duration: '2m' },
        { target: 600, duration: '4m' },
        { target: 1000, duration: '4m' },
        { target: 0, duration: '30s' },
      ],
      exec: 'jsonTest',
    },
  },
  thresholds: {
    // Falhas totais por cenário
    'http_req_failed{scenario:redirect_scenario}': ['rate<0.01'],
    'http_req_failed{scenario:json_scenario}': ['rate<0.01'],
    // Latências (p95/p99) por cenário
    'http_req_duration{scenario:redirect_scenario}': ['p(95)<200', 'p(99)<400'],
    'http_req_duration{scenario:json_scenario}': ['p(95)<300', 'p(99)<600'],
  },
};

// --------- TARGETS ---------
const REDIRECT_URL = 'https://ads.conexao.gru.br/jc8qYZXT?1754856558418';
const JSON_URL = 'https://ads.conexao.gru.br/?type=news&ad_type_4=1&1754856696414';

// --------- EXEC ---------
export function redirectTest() {
  // Não seguir redirecionamento para medir só o 3xx do Go/Nginx
  const res = http.get(REDIRECT_URL, { redirects: 0 });
  check(res, {
    'status 3xx': (r) => r.status >= 300 && r.status < 400,
    'tem header Location': (r) => !!r.headers['Location'] || !!r.headers['location'],
  });
  // Pense-time pequeno (simula browser)
  sleep(0.1);
}

export function jsonTest() {
  const res = http.get(JSON_URL, { headers: { 'Accept': 'application/json' }});
  check(res, {
    'status 200': (r) => r.status === 200,
    'json válido': (r) => {
      try { JSON.parse(r.body); return true; } catch { return false; }
    },
    'tem ads e redirect e static': (r) => {
      try {
        const j = JSON.parse(r.body);
        return Array.isArray(j.ads) && j.redirect && j.static;
      } catch { return false; }
    },
  });
  sleep(0.1);
}
