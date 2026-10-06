// 2回目以降の起動を速くするためのService Worker。
//
// GitHub Pagesのキャッシュは10分ほどで切れるので、そのままでは訪れるたびに
// game.wasm(約18MB)や素材を取り直す。ここでは一度取得したファイルを
// 中身のハッシュ(ASSET_MANIFEST)つきで保存しておき、ハッシュが同じ間は
// ネットワークに行かずに返す。素材を差し替えるとハッシュが変わるので、
// 変わったファイルだけ取り直す(変わっていないファイルはデプロイをまたいで使い回す)。
//
// ページ(index.html)とゲーム本体は、ファイルを「パス?v=ハッシュ」で頼む
// (ハッシュはindex.htmlに書き込まれた同じ一覧から取る)。ここではvが自分の
// 一覧のハッシュと一致するときだけ保存済みのものを返し、違えば素通しする。
// デプロイ直後で古いService Workerがまだ動いていても、新しいページが頼んだ
// 新しい版を、古い版の保存済みファイルで返してしまうことはない。
//
// ASSET_MANIFESTとBUILD_IDは、デプロイ時にtools/genswmanifestが書き込む。
// 書き込まれていない(ローカルの確認用サーバー)ときは、index.htmlがこの
// Service Workerを登録しない。

const BUILD_ID = "__BUILD_ID__";
// 公開フォルダからの相対パス → 中身のハッシュ。
const ASSET_MANIFEST = "__ASSET_MANIFEST__";
const CACHE_NAME = "game-files";

const scopePath = new URL(self.registration.scope).pathname;

// relPath はリクエストのURLを、公開フォルダからの相対パス(デコード済み)にする。
function relPath(url) {
  if (!url.pathname.startsWith(scopePath)) {
    return null;
  }
  try {
    return decodeURIComponent(url.pathname.slice(scopePath.length));
  } catch (e) {
    return null;
  }
}

// cacheKey はpathのファイルを保存するときの鍵。ハッシュを含めるので、
// 中身が変わったファイルは別の鍵になる。
function cacheKey(path, hash) {
  return new URL(scopePath + path + "?v=" + hash, self.location.origin).href;
}

self.addEventListener("install", () => {
  // 新しい版はすぐに有効にする(古い版のまま次の訪問まで待たせない)。
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    // 今の版の一覧に無い(中身が変わった・消えた)ファイルを捨てる。
    const keep = new Set(Object.entries(ASSET_MANIFEST).map(([p, h]) => cacheKey(p, h)));
    const cache = await caches.open(CACHE_NAME);
    for (const req of await cache.keys()) {
      if (!keep.has(req.url) && req.url !== indexKey()) {
        await cache.delete(req);
      }
    }
    await self.clients.claim();
  })());
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") {
    return;
  }
  const url = new URL(req.url);
  if (url.origin !== self.location.origin) {
    return;
  }
  if (req.mode === "navigate") {
    event.respondWith(networkFirstIndex(req));
    return;
  }
  const path = relPath(url);
  const hash = path !== null && Object.prototype.hasOwnProperty.call(ASSET_MANIFEST, path) ? ASSET_MANIFEST[path] : null;
  // 版の指定が無い・自分の一覧と違う版の要求は、保存せずそのまま取りに行く。
  if (hash === null || url.searchParams.get("v") !== hash) {
    return;
  }
  event.respondWith(cacheFirst(req, cacheKey(path, hash)));
});

async function cacheFirst(req, key) {
  const cache = await caches.open(CACHE_NAME);
  const hit = await cache.match(key);
  if (hit) {
    return hit;
  }
  const resp = await fetch(req);
  if (resp.ok && resp.status === 200) {
    // 保存は裏で行い、ゲームには先に返す。
    cache.put(key, resp.clone()).catch(() => {});
  }
  return resp;
}

// ページ本体(index.html)は新しい版に気づけるよう、常にネットワークを先に見る。
// つながらないときだけ、前回保存したものを返す。
async function networkFirstIndex(req) {
  const cache = await caches.open(CACHE_NAME);
  const key = indexKey();
  try {
    const resp = await fetch(req);
    if (resp.ok) {
      cache.put(key, resp.clone()).catch(() => {});
    }
    return resp;
  } catch (e) {
    const hit = await cache.match(key);
    if (hit) {
      return hit;
    }
    throw e;
  }
}

// indexKey は前回のindex.htmlを保存しておく鍵(Cache APIはURLの#以下を
// 無視するので、クエリで区別する)。
function indexKey() {
  return new URL(scopePath + "?sw=index", self.location.origin).href;
}
