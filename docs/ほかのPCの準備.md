# ほかのPCの準備(1台ずつ・最初に1回だけ)

## 1. GitHub Desktop を入れて、フォルダを作る

1. `desktop.github.com` から GitHub Desktop を入れる
2. メインPCと**同じGitHubアカウント**でサインイン
3. 「**Clone a repository**」→ `go-sync-project` を選ぶ →「Clone」
4. できたフォルダが**作業フォルダ**。GitHub Desktop の「**Show in Explorer**」でいつでも開ける

## 2. Tiled でプロジェクトを開く

1. Tiled を起動
2. 「**プロジェクト**」→「**Open Project...**」(英語のまま出る)→ 作業フォルダの `rpg.tiled-project`
3. 左にマップの一覧が出れば成功(次からは自動で開く)

## 3. 試し遊びで確かめる

1. 左の一覧から `school_1.tmj` を開く
2. **F5** を押す → ゲームが始まれば準備完了
   - 最初の1回だけ **1〜2分** かかる(2回目からは速い)

---

## 4. このPCに前からあるマップを移す(あるPCだけ)

1. マップを Tiled で開いていたら、閉じる
2. エクスプローラーでコピーする
   - マップ(`.tmj`)→ 作業フォルダの `assets\maps`
   - そのマップで使っているタイル画像(`.png`)→ 作業フォルダの `assets\images\tiles`
   - ファイル名は**英小文字**にする。`school_1`・`water_b` と同じ名前は使わない
3. GitHub Desktop で、下に「前のマップを追加」と書いて **Commit to main → Push origin**
4. Claude に「**前のマップを送った**」と伝える
   - タイルセットの付け直し・しかけのクラス化・ドアなどを直してもらう
   - 直るまでは自動チェックで止まるので、Web版は壊れない
5. 直し終わったと言われたら、GitHub Desktop で **Fetch origin → Pull origin**
6. Tiled で開いて F5 で確かめる

**直してもらっている間は、そのマップを Tiled で触らない。**

---

## 準備が済んだら:毎日の流れ

**作業を始めるとき**
1. Tiled を閉じる
2. GitHub Desktop で **Fetch origin → Pull origin**

**作業が終わったとき**
1. GitHub Desktop の左下に一言書く
2. **Commit to main → Push origin**

**同じマップを2台以上で同時に直さない。**
