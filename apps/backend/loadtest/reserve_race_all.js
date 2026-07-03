import http from 'k6/http';
import { Counter } from 'k6/metrics';

// Combined version of reserve_race.js: fires against BOTH categories at once,
// in the same run, to prove total successes never exceed total real stock
// (100 VIP + 400 Standard = 500 tickets) -- matching the product goal stated
// in README.md (5,000 concurrent users, 500 tickets, zero overselling).
//
// VU counts default to the same 1:4 ratio as real seeded stock (100:400) so
// contention per category is proportional to its actual inventory.
//
// Usage:
//   k6 run --env BASE_URL=http://localhost:8080 --env VIP_VUS=1000 --env STANDARD_VUS=4000 reserve_race_all.js

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const VIP_VUS = parseInt(__ENV.VIP_VUS || '1000', 10);
const STANDARD_VUS = parseInt(__ENV.STANDARD_VUS || '4000', 10);

export const options = {
  scenarios: {
    reserve_vip: {
      executor: 'per-vu-iterations',
      vus: VIP_VUS,
      iterations: 1,
      maxDuration: '60s',
      exec: 'reserveVip',
    },
    reserve_standard: {
      executor: 'per-vu-iterations',
      vus: STANDARD_VUS,
      iterations: 1,
      maxDuration: '60s',
      exec: 'reserveStandard',
    },
  },
};

const successVip = new Counter('reserve_success_vip');
const soldoutVip = new Counter('reserve_soldout_vip');
const successStandard = new Counter('reserve_success_standard');
const soldoutStandard = new Counter('reserve_soldout_standard');
const unexpected = new Counter('reserve_unexpected');

function doReserve(category, successCounter, soldoutCounter) {
  const res = http.post(
    `${BASE_URL}/api/v1/tickets/reserve`,
    JSON.stringify({ category }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (res.status === 201) {
    successCounter.add(1);
  } else if (res.status === 409) {
    let code = '';
    try {
      code = JSON.parse(res.body).error.code;
    } catch (e) {
      // ignore parse failure, fall through to unexpected classification below
    }
    if (code === 'TICKET_UNAVAILABLE') {
      soldoutCounter.add(1);
    } else {
      unexpected.add(1);
      console.error(`unexpected 409 body: ${res.body}`);
    }
  } else {
    unexpected.add(1);
    console.error(`unexpected status ${res.status}: ${res.body}`);
  }
}

export function reserveVip() {
  doReserve('VIP', successVip, soldoutVip);
}

export function reserveStandard() {
  doReserve('Standard', successStandard, soldoutStandard);
}
