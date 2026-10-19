const test = require('node:test');
const assert = require('node:assert/strict');
const { loginResult, setup, plain, tick, deferred } = require('./helpers.cjs');

test('backend login fields normalize without losing a 64-bit user ID', () => {
  const { session, localStorage } = setup();
  session.saveSession(loginResult);
  assert.deepEqual(plain(session.getSession()), {
    user_id: loginResult.userID,
    user_name: 'zed',
    token: 'access-old',
    refresh_token: 'refresh-old'
  });
  assert.equal(typeof session.getSession().user_id, 'string');
  assert.equal(session.hasSession(), true);
  assert.equal(JSON.parse(localStorage.getItem('loginResult')).user_id, loginResult.userID);
});

test('corrupted localStorage cannot crash startup or create a false login', () => {
  const { session, localStorage } = setup(undefined, { loginResult: '{broken json' });
  assert.equal(session.hasSession(), false);
  assert.equal(localStorage.getItem('loginResult'), null);
});

test('unsafe numeric IDs and literal undefined tokens are rejected', () => {
  const { session } = setup();
  assert.equal(session.normalizeSession({ ...loginResult, userID: 9223372036854776000 }), null);
  assert.equal(session.normalizeSession({ ...loginResult, accessTokne: 'undefined' }), null);
});

test('a persisted canonical session survives browser reload', () => {
  const first = setup();
  first.session.saveSession(loginResult);
  const next = setup(undefined, { loginResult: first.localStorage.getItem('loginResult') });
  assert.equal(next.session.getSession().token, 'access-old');
  assert.equal(next.session.getSession().user_id, loginResult.userID);
});

test('protected requests receive exactly one Bearer prefix', async () => {
  const { api, session, requests } = setup();
  session.saveSession(loginResult);
  await api.get('/community');
  assert.equal(requests[0].headers.Authorization, 'Bearer access-old');
});

test('anonymous requests never send Bearer undefined', async () => {
  const { api, requests } = setup();
  await api.get('/posts2');
  assert.equal(requests[0].headers.Authorization, undefined);
});

test('login and signup never inherit a previous account Bearer token', async () => {
  const { api, session, requests } = setup();
  session.saveSession(loginResult);
  await api.post('/login', { username: 'zed', password: 'test-only' });
  await api.post('/signup', { username: 'zed', password: 'test-only', repassword: 'test-only' });
  assert.ok(requests.every(request => request.headers.Authorization === undefined));
});

test('an invalid response body is rejected instead of being treated as business success', async () => {
  const { api } = setup(async () => 'rate limit ....');
  await assert.rejects(api.get('/posts2'), /服务响应异常/);
});

test('a transport failure remains a rejected promise', async () => {
  const failure = new Error('offline');
  const { api } = setup(async () => { throw failure; });
  await assert.rejects(api.get('/posts2'), error => error === failure);
});

test('parallel expired requests share one refresh and retry with the new access token', async () => {
  const refresh = deferred();
  let refreshCalls = 0;
  const { api, session, requests } = setup(async request => {
    if (request.url === '/refresh') {
      refreshCalls += 1;
      await refresh.promise;
      assert.deepEqual(plain(request.data), { access_token: 'access-old', refresh_token: 'refresh-old' });
      assert.equal(request.headers.Authorization, undefined);
      return { code: 1000, data: { access_token: 'access-new', refresh_token: 'refresh-new' } };
    }
    return request.headers.Authorization === 'Bearer access-new'
      ? { code: 1000, data: { ok: request.url } }
      : { code: 1006, msg: '无效的token' };
  });
  session.saveSession(loginResult);
  const first = api.get('/community');
  const second = api.get('/posts2');
  await tick();
  assert.equal(refreshCalls, 1);
  refresh.resolve();
  const results = await Promise.all([first, second]);
  assert.deepEqual(results.map(result => result.code), [1000, 1000]);
  assert.equal(refreshCalls, 1);
  assert.equal(requests.filter(request => request.url !== '/refresh').length, 4);
  assert.equal(session.getSession().token, 'access-new');
  assert.equal(session.getSession().refresh_token, 'refresh-new');
});

test('a second invalid-token response stops retrying and clears the invalid session', async () => {
  let refreshCalls = 0;
  let resourceCalls = 0;
  const { api, session } = setup(async request => {
    if (request.url === '/refresh') {
      refreshCalls += 1;
      return { code: 1000, data: { access_token: 'still-invalid', refresh_token: 'refresh-new' } };
    }
    resourceCalls += 1;
    return { code: 1006, msg: '无效的token' };
  });
  session.saveSession(loginResult);
  const result = await api.get('/community');
  assert.equal(result.code, 1006);
  assert.equal(refreshCalls, 1);
  assert.equal(resourceCalls, 2);
  assert.equal(session.hasSession(), false);
});

test('a session replaced by another device logs out immediately without refreshing', async () => {
  const { api, session, requests, localStorage } = setup(async () => ({ code: 1009, msg: '请重新登录' }));
  session.saveSession(loginResult);
  const result = await api.get('/community');
  assert.equal(result.code, 1009);
  assert.equal(requests.length, 1);
  assert.equal(session.hasSession(), false);
  assert.equal(localStorage.getItem('loginResult'), null);
});

test('an invalid refresh token clears the session and does not retry the resource', async () => {
  const { api, session, requests } = setup(async () => ({ code: 1006, msg: '无效的token' }));
  session.saveSession(loginResult);
  const result = await api.get('/community');
  assert.equal(result.code, 1006);
  assert.equal(requests.length, 2);
  assert.equal(session.hasSession(), false);
});

test('a service error preserves the login and never tries a token refresh', async () => {
  const { api, session, requests } = setup(async () => ({ code: 1005, msg: '服务繁忙' }));
  session.saveSession(loginResult);
  const result = await api.get('/community');
  assert.equal(result.code, 1005);
  assert.equal(session.hasSession(), true);
  assert.equal(requests.length, 1);
});

test('a temporary refresh service failure preserves the current session', async () => {
  const { api, session } = setup(async request => request.url === '/refresh'
    ? { code: 1005, msg: '服务繁忙' }
    : { code: 1006, msg: '无效的token' });
  session.saveSession(loginResult);
  await assert.rejects(api.get('/community'), /服务繁忙/);
  assert.equal(session.getSession().token, 'access-old');
});

test('a delayed expired response reuses a refresh already completed by another request', async () => {
  const delayed = deferred();
  let refreshCalls = 0;
  const { api, session } = setup(async request => {
    if (request.url === '/refresh') {
      refreshCalls += 1;
      return { code: 1000, data: { access_token: 'access-new', refresh_token: 'refresh-new' } };
    }
    if (request.headers.Authorization === 'Bearer access-new') return { code: 1000, data: 'ok' };
    if (request.url === '/slow') await delayed.promise;
    return { code: 1006, msg: '无效的token' };
  });
  session.saveSession(loginResult);
  const slow = api.get('/slow');
  await tick();
  assert.equal((await api.get('/community')).code, 1000);
  delayed.resolve();
  assert.equal((await slow).code, 1000);
  assert.equal(refreshCalls, 1);
});

test('a late kicked-out response from an old request cannot clear a newer login', async () => {
  const delayed = deferred();
  const { api, session } = setup(async () => {
    await delayed.promise;
    return { code: 1009, msg: '请重新登录' };
  });
  session.saveSession(loginResult);
  const pending = api.get('/community');
  await tick();
  session.saveSession({ ...loginResult, accessTokne: 'another-login', refreshToken: 'another-refresh' });
  delayed.resolve();
  assert.equal((await pending).code, 1009);
  assert.equal(session.getSession().token, 'another-login');
});

test('logging out while refresh is pending cannot resurrect the session', async () => {
  const refresh = deferred();
  const { api, session } = setup(async request => {
    if (request.url === '/refresh') {
      await refresh.promise;
      return { code: 1000, data: { access_token: 'late-access', refresh_token: 'late-refresh' } };
    }
    return { code: 1006, msg: '无效的token' };
  });
  session.saveSession(loginResult);
  const pending = api.get('/community');
  await tick();
  session.clearSession();
  refresh.resolve();
  await assert.rejects(pending, /登录状态已变更/);
  assert.equal(session.hasSession(), false);
});

test('a late refresh response cannot overwrite a newer login', async () => {
  const refresh = deferred();
  const { api, session } = setup(async request => {
    if (request.url === '/refresh') {
      await refresh.promise;
      return { code: 1000, data: { access_token: 'late-access', refresh_token: 'late-refresh' } };
    }
    return { code: 1006, msg: '无效的token' };
  });
  session.saveSession(loginResult);
  const pending = api.get('/community');
  await tick();
  session.saveSession({ ...loginResult, accessTokne: 'another-login', refreshToken: 'another-refresh' });
  refresh.resolve();
  await assert.rejects(pending, /登录状态已变更/);
  assert.equal(session.getSession().token, 'another-login');
  assert.equal(session.getSession().refresh_token, 'another-refresh');
});
