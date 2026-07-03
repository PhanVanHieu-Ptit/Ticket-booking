import http from 'k6/http';
import { Counter } from 'k6/metrics';

// Fires exactly one POST /tickets/reserve per VU, all VUs starting together,
// to prove the hold-ticket endpoint cannot hand out more holds than the
// category's actual stock under real concurrency.
//
// Usage:
//   k6 run --env BASE_URL=http://localhost:8080 --env CATEGORY=VIP --env VUS=500 reserve_race.js

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const CATEGORY = __ENV.CATEGORY || 'VIP';
const VUS = parseInt(__ENV.VUS || '500', 10);

export const options = {
  scenarios: {
    reserve_race: {
      executor: 'per-vu-iterations',
      vus: VUS,
      iterations: 1,
      maxDuration: '60s',
    },
  },
};

const reserveSuccess = new Counter('reserve_success');
const reserveSoldOut = new Counter('reserve_soldout');
const reserveUnexpected = new Counter('reserve_unexpected');

export default function () {
  // No cookie is set, so SessionMiddleware mints a brand new session_id for
  // this request -- each VU is a distinct, unauthenticated "buyer".
  const res = http.post(
    `${BASE_URL}/api/v1/tickets/reserve`,
    JSON.stringify({ category: CATEGORY }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (res.status === 201) {
    reserveSuccess.add(1);
  } else if (res.status === 409) {
    let code = '';
    try {
      code = JSON.parse(res.body).error.code;
    } catch (e) {
      // ignore parse failure, fall through to unexpected classification below
    }
    if (code === 'TICKET_UNAVAILABLE') {
      reserveSoldOut.add(1);
    } else {
      reserveUnexpected.add(1);
      console.error(`unexpected 409 body: ${res.body}`);
    }
  } else {
    reserveUnexpected.add(1);
    console.error(`unexpected status ${res.status}: ${res.body}`);
  }
}
