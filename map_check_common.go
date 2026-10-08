package main

// mapIssue はマップのチェック(map_check.go)で見つかったまちがい1つ。
type mapIssue struct {
	Path string // マップ(またはタイルセット)のファイル
	Msg  string
}

func (i mapIssue) String() string { return i.Path + ": " + i.Msg }
