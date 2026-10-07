# マップの受け渡し(メール用)。
#   send    : assets のマップ・タイルセット・タイル画像を「マップ一式.zip」にまとめて
#             デスクトップに置く。
#   receive : ダウンロードフォルダ(またはデスクトップ)にある一番新しい
#             「マップ一式*.zip」を、このフォルダの assets に取り込む。
#             このPCのファイルの方が新しいときは上書きしない(まだ送っていない
#             作業を消さないため)。
# どちらのPCでも、このフォルダ(assets の1つ上)の bat から呼ばれる。
param(
    [Parameter(Mandatory = $true)][ValidateSet('send', 'receive')][string]$Mode,
    # テスト用: ZIPの場所を直接指定する(指定するとエクスプローラーも開かない)
    [string]$ZipPath
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$zipName = 'マップ一式.zip'
$syncDirs = @('assets\maps', 'assets\tilesets', 'assets\images\tiles')
# 相手のPCが最初に展開したときに使えるよう、bat とこのスクリプト、Tiledの
# プロジェクトファイル(一覧から選ぶクラス・F5で試し遊び)も入れる。
$guide = 'docs\マップ作りガイド.md'
$toolFiles = @('マップを送る.bat', 'マップを受け取る.bat', 'tools\mapsync\mapsync.ps1', 'rpg.tiled-project', 'tools\playmap\play.bat', $guide)
# 受け取るときに取り込むもの(動いている bat とこのスクリプトは置き換えない)。
function Test-Receivable([string]$entry) {
    return $entry.StartsWith('assets/') -or $entry -eq 'rpg.tiled-project' -or $entry -eq 'tools/playmap/play.bat' -or $entry -eq $guide.Replace('\', '/')
}

function Send-Maps {
    $zipPath = $ZipPath
    if (-not $zipPath) { $zipPath = Join-Path ([Environment]::GetFolderPath('Desktop')) $zipName }
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }

    $zip = [System.IO.Compression.ZipFile]::Open($zipPath, 'Create')
    $count = 0
    try {
        $files = @()
        foreach ($d in $syncDirs) {
            $dir = Join-Path $root $d
            if (Test-Path $dir) { $files += Get-ChildItem $dir -Recurse -File }
        }
        foreach ($t in $toolFiles) {
            $p = Join-Path $root $t
            if (Test-Path $p) { $files += Get-Item $p }
        }
        foreach ($f in $files) {
            $rel = $f.FullName.Substring($root.Length).TrimStart('\').Replace('\', '/')
            [void][System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, $f.FullName, $rel)
            if ($rel.StartsWith('assets/')) { $count++ }
        }
    }
    finally { $zip.Dispose() }

    Write-Host ""
    Write-Host "「$zipName」を作りました(ファイル $count 個): $(Split-Path $zipPath -Parent)" -ForegroundColor Green
    Write-Host "このZIPをメールに添付して送ってください。"
    if (-not $ZipPath) { Start-Process explorer.exe "/select,`"$zipPath`"" }
}

function Find-Zip {
    if ($ZipPath) { return Get-Item $ZipPath }
    $places = @()
    try { $places += (New-Object -ComObject Shell.Application).NameSpace('shell:Downloads').Self.Path } catch {}
    $places += Join-Path $env:USERPROFILE 'Downloads'
    $places += [Environment]::GetFolderPath('Desktop')
    $found = foreach ($p in ($places | Select-Object -Unique)) {
        if ($p -and (Test-Path $p)) { Get-ChildItem $p -Filter 'マップ一式*.zip' -File }
    }
    $found | Sort-Object LastWriteTime -Descending | Select-Object -First 1
}

function Receive-Maps {
    $zipFile = Find-Zip
    if (-not $zipFile) {
        Write-Host ""
        Write-Host "「$zipName」が見つかりません。" -ForegroundColor Red
        Write-Host "メールの添付ファイルを、ダウンロードフォルダかデスクトップに保存してから、もう一度実行してください。"
        return
    }
    Write-Host ""
    Write-Host "取り込むファイル: $($zipFile.FullName)"
    Write-Host ""

    $new = @(); $updated = @(); $kept = @()
    $zip = [System.IO.Compression.ZipFile]::OpenRead($zipFile.FullName)
    try {
        foreach ($e in $zip.Entries) {
            if (-not (Test-Receivable $e.FullName) -or $e.FullName.EndsWith('/')) { continue }
            $dest = Join-Path $root ($e.FullName.Replace('/', '\'))
            $zipTime = $e.LastWriteTime.DateTime
            if (Test-Path $dest) {
                $local = Get-Item $dest
                if ($local.Length -eq $e.Length -and [Math]::Abs(($local.LastWriteTime - $zipTime).TotalSeconds) -le 2) {
                    continue
                }
                # このPCの方が新しい = まだ送っていない作業がある。上書きしない。
                if ($local.LastWriteTime -gt $zipTime.AddSeconds(2)) {
                    $kept += $e.FullName
                    continue
                }
                $updated += $e.FullName
            }
            else {
                $new += $e.FullName
            }
            $dir = Split-Path $dest -Parent
            if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
            [System.IO.Compression.ZipFileExtensions]::ExtractToFile($e, $dest, $true)
            (Get-Item $dest).LastWriteTime = $zipTime
        }
    }
    finally { $zip.Dispose() }

    foreach ($f in $new) { Write-Host "  追加    $f" -ForegroundColor Green }
    foreach ($f in $updated) { Write-Host "  上書き  $f" -ForegroundColor Cyan }
    foreach ($f in $kept) { Write-Host "  そのまま $f (このPCの方が新しいため上書きしませんでした)" -ForegroundColor Yellow }
    Write-Host ""
    if ($new.Count + $updated.Count -eq 0) {
        Write-Host "新しく取り込むファイルはありませんでした。"
    }
    else {
        Write-Host "取り込みが終わりました(追加 $($new.Count) 個、上書き $($updated.Count) 個)。" -ForegroundColor Green
    }
    if ($kept.Count -gt 0) {
        Write-Host "黄色のファイルは、このPCで直したものの方が新しいので残しました。" -ForegroundColor Yellow
        Write-Host "相手の変更を使いたいときは、どちらの内容を残すか決めてから手で置き換えてください。" -ForegroundColor Yellow
    }
}

if ($Mode -eq 'send') { Send-Maps } else { Receive-Maps }
