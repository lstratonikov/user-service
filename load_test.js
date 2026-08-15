import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

// SharedArray загружает патроны один раз и шарит между всеми VUs
const bullets = new SharedArray('bullets', function() {
  return JSON.parse(open('/tmp/bullets.json'));
});

export const options = {
  stages: [
    { duration: '1m', target: 5000 },  // разгон
    { duration: '2m', target: 5000 },  // пик
    { duration: '10s', target: 0 },    // спуск
  ], 
};

export function setup() {
  http.post('http://localhost:8080/admin/test-start');
}

export function teardown() {
  http.post('http://localhost:8080/admin/test-stop');
}

const BASE_URL = 'http://localhost:8080';

function registerUser() {
  const res = http.post(`${BASE_URL}/user/register`, JSON.stringify({
    name: `User ${Math.random()}`,
    email: `user_${Math.random()}@test.com`,
    phone: `+7999${Math.floor(Math.random() * 9000000) + 1000000}`,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'register_user' },
  });

  if (res.status !== 200) {
    console.error(`ERROR register status=${res.status}: ${res.body}`);
  }

  check(res, {
    'register: status 200': (r) => r.status === 200,
    'register: has id': (r) => {
      if (r.status !== 200) return false;
      try { return JSON.parse(r.body).id > 0; } catch { return false; }
    },
  });

  if (res.status === 200) {
    try { return JSON.parse(res.body).id; } catch { return null; }
  }
  return null;
}

function getUser(bullet) {
  const res = http.get(`${BASE_URL}/user/get?id=${bullet.id}`, {
    tags: { name: 'get_user' },
  });

  if (res.status !== 200) {
    console.error(`ERROR get id=${bullet.id} status=${res.status}: ${res.body}`);
  }

  check(res, { 'get: status 200': (r) => r.status === 200 });
}

function updateUser(bullet) {
  const res = http.patch(`${BASE_URL}/user/update`, JSON.stringify({
    id: bullet.id,
    phone: `+7999${Math.floor(Math.random() * 9000000) + 1000000}`,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'update_user' },
  });

  if (res.status !== 200) {
    console.error(`ERROR update id=${bullet.id} status=${res.status}: ${res.body}`);
  }

  check(res, { 'update: status 200': (r) => r.status === 200 });
}

function updateStatus(bullet) {
  const newStatus = bullet.status_id === 1 ? 2 : 1;

  const res = http.patch(`${BASE_URL}/user/updateStatus`, JSON.stringify({
    id: bullet.id,
    status_id: newStatus,
  }), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'update_status' },
  });

  if (res.status !== 200) {
    console.error(`ERROR status id=${bullet.id} status=${res.status}: ${res.body}`);
  }

  check(res, { 'status: status 200': (r) => r.status === 200 });
}

function deleteUser(id) {
  const res = http.request('DELETE', `${BASE_URL}/user/delete?id=${id}`, null, {
    tags: { name: 'delete_user' },
  });

  if (res.status !== 200) {
    console.error(`ERROR delete id=${id} status=${res.status}: ${res.body}`);
  }

  check(res, { 'delete: status 200': (r) => r.status === 200 });
}

export default function () {
  const bullet = bullets[Math.floor(Math.random() * bullets.length)];
  const rand = Math.random();

  if (rand < 0.2) {
    // 20% — register
    registerUser();
  } else if (rand < 0.4) {
    // 20% — get
    getUser(bullet);
  } else if (rand < 0.6) {
    // 20% — update
    updateUser(bullet);
  } else if (rand < 0.8) {
    // 20% — updateStatus
    updateStatus(bullet);
  } else {
    // 20% — delete
    deleteUser(bullet.id);
  }
}

export function handleSummary(data) {
  const rps = data.metrics.http_reqs.values.rate;
  const p95 = data.metrics.http_req_duration.values['p(95)'];
  const errors = data.metrics.http_req_failed.values.rate * 100;

  let recommendation = '';
  if (errors > 1) {
    recommendation = `⚠️  Много ошибок (${errors.toFixed(1)}%). Снизь RPS до ${Math.floor(rps * 0.7)}`;
  } else if (p95 > 500) {
    recommendation = `⚠️  p95 высокий (${p95.toFixed(0)}ms). Оптимальный RPS: ${Math.floor(rps * 0.8)}`;
  } else {
    recommendation = `✅ Сервис держит нагрузку. Можно увеличить до ${Math.floor(rps * 1.3)} RPS`;
  }

  console.log('\n════════════════════════════════');
  console.log(`RPS:          ${rps.toFixed(0)}`);
  console.log(`p95:          ${p95.toFixed(0)}ms`);
  console.log(`Ошибки:       ${errors.toFixed(1)}%`);
  console.log(`Рекомендация: ${recommendation}`);
  console.log('════════════════════════════════\n');

  return {};
}