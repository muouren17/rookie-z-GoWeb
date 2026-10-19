const test = require('node:test');
const assert = require('node:assert/strict');
const { loadModule, plain, deferred, tick } = require('./helpers.cjs');

function component(name) {
  return loadModule('views/' + name + '.vue').default;
}

function context(view, extra = {}) {
  return { ...view.data(), ...view.methods, ...extra };
}

test('registration sends backend repassword and navigates after success', async () => {
  const signup = component('SignUp');
  const requests = [];
  const destinations = [];
  const page = context(signup, {
    username: 'test-user', password: 'test-password', confirm_password: 'test-password',
    $axios: async request => { requests.push(request); return { code: 1000 }; },
    $router: { push: async destination => destinations.push(destination) }
  });
  await page.submit();
  assert.equal(requests[0].url, '/signup');
  assert.deepEqual(plain(requests[0].data), {
    username: 'test-user', password: 'test-password', repassword: 'test-password'
  });
  assert.equal(Object.hasOwn(requests[0].data, 'confirm_password'), false);
  assert.equal(destinations[0].name, 'Login');
  assert.equal(page.submitted, false);
});

test('mismatched registration passwords do not send an HTTP request', async () => {
  const signup = component('SignUp');
  let sent = false;
  const page = context(signup, {
    username: 'test-user', password: 'one-password', confirm_password: 'another-password',
    $axios: async () => { sent = true; }
  });
  await page.submit();
  assert.equal(sent, false);
  assert.match(page.error, /密码不一致/);
});

test('post detail reads a nested community safely before and after loading', () => {
  const detail = component('Content');
  assert.deepEqual(plain(detail.computed.community.call({ post: {} })), {});
  const nested = { id: 1, name: 'Go', introduction: 'Go community' };
  assert.equal(detail.computed.community.call({ post: { community: nested } }), nested);
  assert.equal(detail.computed.communityCreatedAt.call({ community: {} }), '');
});

test('post detail requests preserve a 64-bit post ID exactly', async () => {
  const detail = component('Content');
  const postID = '9223372036854775806';
  const requests = [];
  const page = context(detail, {
    $route: { params: { id: postID } },
    $axios: async request => {
      requests.push(request);
      return { code: 1000, data: { id: postID, community: { name: 'Go' } } };
    }
  });
  await page.getPostDetail();
  assert.equal(requests[0].url, '/post/' + postID);
  assert.equal(page.post.id, postID);
  assert.equal(page.loading, false);
});

test('a late previous detail response cannot replace the currently selected post', async () => {
  const detail = component('Content');
  const old = deferred();
  const page = context(detail, {
    $route: { params: { id: '9007199254740993' } },
    $axios: async request => {
      if (request.url.endsWith('9007199254740993')) {
        await old.promise;
        return { code: 1000, data: { id: '9007199254740993', title: 'old' } };
      }
      return { code: 1000, data: { id: '9007199254740995', title: 'new' } };
    }
  });
  const first = page.getPostDetail();
  await tick();
  page.$route.params.id = '9007199254740995';
  await page.getPostDetail();
  old.resolve();
  await first;
  assert.equal(page.post.id, '9007199254740995');
  assert.equal(page.post.title, 'new');
});

test('publishing sends numeric community_id without a client-supplied author ID', async () => {
  const publish = component('Publish');
  const requests = [];
  const page = context(publish, {
    title: 'Test title', content: 'Test content', selectCommunity: { id: 1, name: 'Go' },
    $axios: async request => { requests.push(request); return { code: 1000 }; },
    $router: { push: async () => {} }
  });
  await page.submit();
  assert.equal(requests[0].url, '/post');
  assert.deepEqual(plain(requests[0].data), { title: 'Test title', content: 'Test content', community_id: 1 });
  assert.equal(typeof requests[0].data.community_id, 'number');
  assert.equal(Object.hasOwn(requests[0].data, 'author_id'), false);
});

test('the frontend uses hash routes so a page reload stays on the backend root', () => {
  function Router(options) { this.options = options; }
  Router.prototype.push = () => Promise.resolve();
  const imports = {
    vue: { default: { use() {} } },
    'vue-router': { default: Router }
  };
  for (const view of ['Home', 'Content', 'Publish', 'Login', 'SignUp']) {
    imports['../views/' + view + '.vue'] = { default: {} };
  }
  const router = loadModule('router/index.js', imports, { process: { env: { BASE_URL: '/' } } }).default;
  assert.equal(router.options.mode, 'hash');
  assert.ok(router.options.routes.some(route => route.path === '/post/:id'));
  assert.equal(router.options.routes.find(route => route.name === 'Publish').meta.requireAuth, true);
});

test('route guards call next once and preserve the requested page for login', () => {
  function Vue() {}
  Vue.prototype.$mount = () => {};
  Vue.config = {};
  let guard;
  const router = { beforeEach(callback) { guard = callback; } };
  const store = { commit() {}, getters: { isLogin: false } };
  loadModule('main.js', {
    vue: { default: Vue },
    './App.vue': { default: {} },
    './router': { default: router },
    './store': { default: store },
    './service/api': { default() {} }
  });
  const destinations = [];
  guard({ meta: { requireAuth: true }, fullPath: '/publish' }, {}, to => destinations.push(to));
  assert.deepEqual(plain(destinations), [{ name: 'Login', query: { redirect: '/publish' } }]);
  store.getters.isLogin = true;
  destinations.length = 0;
  guard({ name: 'Publish', meta: { requireAuth: true }, fullPath: '/publish' }, {}, to => destinations.push(to));
  assert.equal(destinations.length, 1);
  assert.equal(destinations[0], undefined);
});
