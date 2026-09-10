package main

// stats_config.go
// ────────────────────────────────────────────────────────────
// 味方・敵、両方のステータスをここ1ファイルにまとめて調整できるようにした設定ファイル。
//
// ■味方（PlayerStatsByLevel / PlayerExpToNextByLevel）
//   Lv, EXP, HP, MP, 物理攻撃力, 魔法攻撃力, 物理防御力, 魔法防御力, すばやさ, 運 を持つ。
//   ステータスは全レベル(1〜50)×4人分、すべて「計算式ではなく実際の数値」として直接書き出している。
//   PlayerStatsByLevel[レベル-1][キャラIndex] という配列なので、
//   特定のレベル・特定のキャラだけ数値を変えたい場合は、下の該当行をそのまま書き換えればよい。
//   例）3人目のキャラのレベル10のHPだけ変えたい場合：
//       PlayerStatsByLevel[9][2].HP = 250
//   必要EXPは PlayerExpToNextByLevel[レベル-1] で、全キャラ共通。
//
// ■敵（EnemyStatsByType / BossStatsByType）
//   Lv, EXP, HP, MP, 物理攻撃力, 魔法攻撃力, 物理防御力, 魔法防御力, すばやさ を持つ。
//   （味方と違い「運」は持たないため、敵の攻撃は会心せず、敵に対する回避判定も
//     味方側の運のみで行われる＝stat growth.go 参照）

// PlayerStats はレベルに応じた味方1人分のステータス一式。
type PlayerStats struct {
	HP       int // 最大HP
	MP       int // 最大MP
	PhysAtk  int // 物理攻撃力
	MagicAtk int // 魔法攻撃力
	PhysDef  int // 物理防御力
	MagicDef int // 魔法防御力
	Spd      int // すばやさ
	Luck     int // 運（会心率・回避率に影響）
}

// PlayerStatsByLevel[レベル-1][キャラIndex]：そのレベルでのステータス実値。
// 全50レベル×4人分、すべて手入力の実数値として書き出してあるので、
// 好きな行を直接編集するだけでそのレベル・そのキャラのステータスだけを個別に変更できる。
var PlayerStatsByLevel = [maxPlayerLevel][partySize]PlayerStats{
	{ // Lv1
		{HP: 100, MP: 15, PhysAtk: 15, MagicAtk: 10, PhysDef: 10, MagicDef: 10, Spd: 24, Luck: 5}, // プレイヤー1
		{HP: 70, MP: 40, PhysAtk: 8, MagicAtk: 10, PhysDef: 10, MagicDef: 10, Spd: 26, Luck: 8},   // プレイヤー2
		{HP: 110, MP: 20, PhysAtk: 12, MagicAtk: 10, PhysDef: 10, MagicDef: 10, Spd: 22, Luck: 4}, // プレイヤー3
		{HP: 90, MP: 10, PhysAtk: 13, MagicAtk: 10, PhysDef: 10, MagicDef: 10, Spd: 20, Luck: 10}, // プレイヤー4
	},
	{ // Lv2
		{HP: 112, MP: 19, PhysAtk: 19, MagicAtk: 11, PhysDef: 12, MagicDef: 11, Spd: 25, Luck: 6}, // プレイヤー1
		{HP: 78, MP: 47, PhysAtk: 9, MagicAtk: 14, PhysDef: 11, MagicDef: 13, Spd: 28, Luck: 10},  // プレイヤー2
		{HP: 121, MP: 25, PhysAtk: 15, MagicAtk: 12, PhysDef: 13, MagicDef: 12, Spd: 23, Luck: 5}, // プレイヤー3
		{HP: 99, MP: 14, PhysAtk: 15, MagicAtk: 12, PhysDef: 12, MagicDef: 12, Spd: 23, Luck: 13}, // プレイヤー4
	},
	{ // Lv3
		{HP: 124, MP: 23, PhysAtk: 23, MagicAtk: 12, PhysDef: 14, MagicDef: 12, Spd: 26, Luck: 7},  // プレイヤー1
		{HP: 86, MP: 54, PhysAtk: 10, MagicAtk: 18, PhysDef: 12, MagicDef: 16, Spd: 30, Luck: 12},  // プレイヤー2
		{HP: 132, MP: 30, PhysAtk: 18, MagicAtk: 14, PhysDef: 16, MagicDef: 14, Spd: 24, Luck: 6},  // プレイヤー3
		{HP: 108, MP: 18, PhysAtk: 17, MagicAtk: 14, PhysDef: 14, MagicDef: 14, Spd: 26, Luck: 16}, // プレイヤー4
	},
	{ // Lv4
		{HP: 136, MP: 27, PhysAtk: 27, MagicAtk: 13, PhysDef: 16, MagicDef: 13, Spd: 27, Luck: 8},  // プレイヤー1
		{HP: 94, MP: 61, PhysAtk: 11, MagicAtk: 22, PhysDef: 13, MagicDef: 19, Spd: 32, Luck: 14},  // プレイヤー2
		{HP: 143, MP: 35, PhysAtk: 21, MagicAtk: 16, PhysDef: 19, MagicDef: 16, Spd: 25, Luck: 7},  // プレイヤー3
		{HP: 117, MP: 22, PhysAtk: 19, MagicAtk: 16, PhysDef: 16, MagicDef: 16, Spd: 29, Luck: 19}, // プレイヤー4
	},
	{ // Lv5
		{HP: 148, MP: 31, PhysAtk: 31, MagicAtk: 14, PhysDef: 18, MagicDef: 14, Spd: 28, Luck: 9},  // プレイヤー1
		{HP: 102, MP: 68, PhysAtk: 12, MagicAtk: 26, PhysDef: 14, MagicDef: 22, Spd: 34, Luck: 16}, // プレイヤー2
		{HP: 154, MP: 40, PhysAtk: 24, MagicAtk: 18, PhysDef: 22, MagicDef: 18, Spd: 26, Luck: 8},  // プレイヤー3
		{HP: 126, MP: 26, PhysAtk: 21, MagicAtk: 18, PhysDef: 18, MagicDef: 18, Spd: 32, Luck: 22}, // プレイヤー4
	},
	{ // Lv6
		{HP: 160, MP: 35, PhysAtk: 35, MagicAtk: 15, PhysDef: 20, MagicDef: 15, Spd: 29, Luck: 10}, // プレイヤー1
		{HP: 110, MP: 75, PhysAtk: 13, MagicAtk: 30, PhysDef: 15, MagicDef: 25, Spd: 36, Luck: 18}, // プレイヤー2
		{HP: 165, MP: 45, PhysAtk: 27, MagicAtk: 20, PhysDef: 25, MagicDef: 20, Spd: 27, Luck: 9},  // プレイヤー3
		{HP: 135, MP: 30, PhysAtk: 23, MagicAtk: 20, PhysDef: 20, MagicDef: 20, Spd: 35, Luck: 25}, // プレイヤー4
	},
	{ // Lv7
		{HP: 172, MP: 39, PhysAtk: 39, MagicAtk: 16, PhysDef: 22, MagicDef: 16, Spd: 30, Luck: 11}, // プレイヤー1
		{HP: 118, MP: 82, PhysAtk: 14, MagicAtk: 34, PhysDef: 16, MagicDef: 28, Spd: 38, Luck: 20}, // プレイヤー2
		{HP: 176, MP: 50, PhysAtk: 30, MagicAtk: 22, PhysDef: 28, MagicDef: 22, Spd: 28, Luck: 10}, // プレイヤー3
		{HP: 144, MP: 34, PhysAtk: 25, MagicAtk: 22, PhysDef: 22, MagicDef: 22, Spd: 38, Luck: 28}, // プレイヤー4
	},
	{ // Lv8
		{HP: 184, MP: 43, PhysAtk: 43, MagicAtk: 17, PhysDef: 24, MagicDef: 17, Spd: 31, Luck: 12}, // プレイヤー1
		{HP: 126, MP: 89, PhysAtk: 15, MagicAtk: 38, PhysDef: 17, MagicDef: 31, Spd: 40, Luck: 22}, // プレイヤー2
		{HP: 187, MP: 55, PhysAtk: 33, MagicAtk: 24, PhysDef: 31, MagicDef: 24, Spd: 29, Luck: 11}, // プレイヤー3
		{HP: 153, MP: 38, PhysAtk: 27, MagicAtk: 24, PhysDef: 24, MagicDef: 24, Spd: 41, Luck: 31}, // プレイヤー4
	},
	{ // Lv9
		{HP: 196, MP: 47, PhysAtk: 47, MagicAtk: 18, PhysDef: 26, MagicDef: 18, Spd: 32, Luck: 13}, // プレイヤー1
		{HP: 134, MP: 96, PhysAtk: 16, MagicAtk: 42, PhysDef: 18, MagicDef: 34, Spd: 42, Luck: 24}, // プレイヤー2
		{HP: 198, MP: 60, PhysAtk: 36, MagicAtk: 26, PhysDef: 34, MagicDef: 26, Spd: 30, Luck: 12}, // プレイヤー3
		{HP: 162, MP: 42, PhysAtk: 29, MagicAtk: 26, PhysDef: 26, MagicDef: 26, Spd: 44, Luck: 34}, // プレイヤー4
	},
	{ // Lv10
		{HP: 208, MP: 51, PhysAtk: 51, MagicAtk: 19, PhysDef: 28, MagicDef: 19, Spd: 33, Luck: 14},  // プレイヤー1
		{HP: 142, MP: 103, PhysAtk: 17, MagicAtk: 46, PhysDef: 19, MagicDef: 37, Spd: 44, Luck: 26}, // プレイヤー2
		{HP: 209, MP: 65, PhysAtk: 39, MagicAtk: 28, PhysDef: 37, MagicDef: 28, Spd: 31, Luck: 13},  // プレイヤー3
		{HP: 171, MP: 46, PhysAtk: 31, MagicAtk: 28, PhysDef: 28, MagicDef: 28, Spd: 47, Luck: 37},  // プレイヤー4
	},
	{ // Lv11
		{HP: 220, MP: 55, PhysAtk: 55, MagicAtk: 20, PhysDef: 30, MagicDef: 20, Spd: 34, Luck: 15},  // プレイヤー1
		{HP: 150, MP: 110, PhysAtk: 18, MagicAtk: 50, PhysDef: 20, MagicDef: 40, Spd: 46, Luck: 28}, // プレイヤー2
		{HP: 220, MP: 70, PhysAtk: 42, MagicAtk: 30, PhysDef: 40, MagicDef: 30, Spd: 32, Luck: 14},  // プレイヤー3
		{HP: 180, MP: 50, PhysAtk: 33, MagicAtk: 30, PhysDef: 30, MagicDef: 30, Spd: 50, Luck: 40},  // プレイヤー4
	},
	{ // Lv12
		{HP: 232, MP: 59, PhysAtk: 59, MagicAtk: 21, PhysDef: 32, MagicDef: 21, Spd: 35, Luck: 16},  // プレイヤー1
		{HP: 158, MP: 117, PhysAtk: 19, MagicAtk: 54, PhysDef: 21, MagicDef: 43, Spd: 48, Luck: 30}, // プレイヤー2
		{HP: 231, MP: 75, PhysAtk: 45, MagicAtk: 32, PhysDef: 43, MagicDef: 32, Spd: 33, Luck: 15},  // プレイヤー3
		{HP: 189, MP: 54, PhysAtk: 35, MagicAtk: 32, PhysDef: 32, MagicDef: 32, Spd: 53, Luck: 43},  // プレイヤー4
	},
	{ // Lv13
		{HP: 244, MP: 63, PhysAtk: 63, MagicAtk: 22, PhysDef: 34, MagicDef: 22, Spd: 36, Luck: 17},  // プレイヤー1
		{HP: 166, MP: 124, PhysAtk: 20, MagicAtk: 58, PhysDef: 22, MagicDef: 46, Spd: 50, Luck: 32}, // プレイヤー2
		{HP: 242, MP: 80, PhysAtk: 48, MagicAtk: 34, PhysDef: 46, MagicDef: 34, Spd: 34, Luck: 16},  // プレイヤー3
		{HP: 198, MP: 58, PhysAtk: 37, MagicAtk: 34, PhysDef: 34, MagicDef: 34, Spd: 56, Luck: 46},  // プレイヤー4
	},
	{ // Lv14
		{HP: 256, MP: 67, PhysAtk: 67, MagicAtk: 23, PhysDef: 36, MagicDef: 23, Spd: 37, Luck: 18},  // プレイヤー1
		{HP: 174, MP: 131, PhysAtk: 21, MagicAtk: 62, PhysDef: 23, MagicDef: 49, Spd: 52, Luck: 34}, // プレイヤー2
		{HP: 253, MP: 85, PhysAtk: 51, MagicAtk: 36, PhysDef: 49, MagicDef: 36, Spd: 35, Luck: 17},  // プレイヤー3
		{HP: 207, MP: 62, PhysAtk: 39, MagicAtk: 36, PhysDef: 36, MagicDef: 36, Spd: 59, Luck: 49},  // プレイヤー4
	},
	{ // Lv15
		{HP: 268, MP: 71, PhysAtk: 71, MagicAtk: 24, PhysDef: 38, MagicDef: 24, Spd: 38, Luck: 19},  // プレイヤー1
		{HP: 182, MP: 138, PhysAtk: 22, MagicAtk: 66, PhysDef: 24, MagicDef: 52, Spd: 54, Luck: 36}, // プレイヤー2
		{HP: 264, MP: 90, PhysAtk: 54, MagicAtk: 38, PhysDef: 52, MagicDef: 38, Spd: 36, Luck: 18},  // プレイヤー3
		{HP: 216, MP: 66, PhysAtk: 41, MagicAtk: 38, PhysDef: 38, MagicDef: 38, Spd: 62, Luck: 52},  // プレイヤー4
	},
	{ // Lv16
		{HP: 280, MP: 75, PhysAtk: 75, MagicAtk: 25, PhysDef: 40, MagicDef: 25, Spd: 39, Luck: 20},  // プレイヤー1
		{HP: 190, MP: 145, PhysAtk: 23, MagicAtk: 70, PhysDef: 25, MagicDef: 55, Spd: 56, Luck: 38}, // プレイヤー2
		{HP: 275, MP: 95, PhysAtk: 57, MagicAtk: 40, PhysDef: 55, MagicDef: 40, Spd: 37, Luck: 19},  // プレイヤー3
		{HP: 225, MP: 70, PhysAtk: 43, MagicAtk: 40, PhysDef: 40, MagicDef: 40, Spd: 65, Luck: 55},  // プレイヤー4
	},
	{ // Lv17
		{HP: 292, MP: 79, PhysAtk: 79, MagicAtk: 26, PhysDef: 42, MagicDef: 26, Spd: 40, Luck: 21},  // プレイヤー1
		{HP: 198, MP: 152, PhysAtk: 24, MagicAtk: 74, PhysDef: 26, MagicDef: 58, Spd: 58, Luck: 40}, // プレイヤー2
		{HP: 286, MP: 100, PhysAtk: 60, MagicAtk: 42, PhysDef: 58, MagicDef: 42, Spd: 38, Luck: 20}, // プレイヤー3
		{HP: 234, MP: 74, PhysAtk: 45, MagicAtk: 42, PhysDef: 42, MagicDef: 42, Spd: 68, Luck: 58},  // プレイヤー4
	},
	{ // Lv18
		{HP: 304, MP: 83, PhysAtk: 83, MagicAtk: 27, PhysDef: 44, MagicDef: 27, Spd: 41, Luck: 22},  // プレイヤー1
		{HP: 206, MP: 159, PhysAtk: 25, MagicAtk: 78, PhysDef: 27, MagicDef: 61, Spd: 60, Luck: 42}, // プレイヤー2
		{HP: 297, MP: 105, PhysAtk: 63, MagicAtk: 44, PhysDef: 61, MagicDef: 44, Spd: 39, Luck: 21}, // プレイヤー3
		{HP: 243, MP: 78, PhysAtk: 47, MagicAtk: 44, PhysDef: 44, MagicDef: 44, Spd: 71, Luck: 61},  // プレイヤー4
	},
	{ // Lv19
		{HP: 316, MP: 87, PhysAtk: 87, MagicAtk: 28, PhysDef: 46, MagicDef: 28, Spd: 42, Luck: 23},  // プレイヤー1
		{HP: 214, MP: 166, PhysAtk: 26, MagicAtk: 82, PhysDef: 28, MagicDef: 64, Spd: 62, Luck: 44}, // プレイヤー2
		{HP: 308, MP: 110, PhysAtk: 66, MagicAtk: 46, PhysDef: 64, MagicDef: 46, Spd: 40, Luck: 22}, // プレイヤー3
		{HP: 252, MP: 82, PhysAtk: 49, MagicAtk: 46, PhysDef: 46, MagicDef: 46, Spd: 74, Luck: 64},  // プレイヤー4
	},
	{ // Lv20
		{HP: 328, MP: 91, PhysAtk: 91, MagicAtk: 29, PhysDef: 48, MagicDef: 29, Spd: 43, Luck: 24},  // プレイヤー1
		{HP: 222, MP: 173, PhysAtk: 27, MagicAtk: 86, PhysDef: 29, MagicDef: 67, Spd: 64, Luck: 46}, // プレイヤー2
		{HP: 319, MP: 115, PhysAtk: 69, MagicAtk: 48, PhysDef: 67, MagicDef: 48, Spd: 41, Luck: 23}, // プレイヤー3
		{HP: 261, MP: 86, PhysAtk: 51, MagicAtk: 48, PhysDef: 48, MagicDef: 48, Spd: 77, Luck: 67},  // プレイヤー4
	},
	{ // Lv21
		{HP: 340, MP: 95, PhysAtk: 95, MagicAtk: 30, PhysDef: 50, MagicDef: 30, Spd: 44, Luck: 25},  // プレイヤー1
		{HP: 230, MP: 180, PhysAtk: 28, MagicAtk: 90, PhysDef: 30, MagicDef: 70, Spd: 66, Luck: 48}, // プレイヤー2
		{HP: 330, MP: 120, PhysAtk: 72, MagicAtk: 50, PhysDef: 70, MagicDef: 50, Spd: 42, Luck: 24}, // プレイヤー3
		{HP: 270, MP: 90, PhysAtk: 53, MagicAtk: 50, PhysDef: 50, MagicDef: 50, Spd: 80, Luck: 70},  // プレイヤー4
	},
	{ // Lv22
		{HP: 352, MP: 99, PhysAtk: 99, MagicAtk: 31, PhysDef: 52, MagicDef: 31, Spd: 45, Luck: 26},  // プレイヤー1
		{HP: 238, MP: 187, PhysAtk: 29, MagicAtk: 94, PhysDef: 31, MagicDef: 73, Spd: 68, Luck: 50}, // プレイヤー2
		{HP: 341, MP: 125, PhysAtk: 75, MagicAtk: 52, PhysDef: 73, MagicDef: 52, Spd: 43, Luck: 25}, // プレイヤー3
		{HP: 279, MP: 94, PhysAtk: 55, MagicAtk: 52, PhysDef: 52, MagicDef: 52, Spd: 83, Luck: 73},  // プレイヤー4
	},
	{ // Lv23
		{HP: 364, MP: 103, PhysAtk: 103, MagicAtk: 32, PhysDef: 54, MagicDef: 32, Spd: 46, Luck: 27}, // プレイヤー1
		{HP: 246, MP: 194, PhysAtk: 30, MagicAtk: 98, PhysDef: 32, MagicDef: 76, Spd: 70, Luck: 52},  // プレイヤー2
		{HP: 352, MP: 130, PhysAtk: 78, MagicAtk: 54, PhysDef: 76, MagicDef: 54, Spd: 44, Luck: 26},  // プレイヤー3
		{HP: 288, MP: 98, PhysAtk: 57, MagicAtk: 54, PhysDef: 54, MagicDef: 54, Spd: 86, Luck: 76},   // プレイヤー4
	},
	{ // Lv24
		{HP: 376, MP: 107, PhysAtk: 107, MagicAtk: 33, PhysDef: 56, MagicDef: 33, Spd: 47, Luck: 28}, // プレイヤー1
		{HP: 254, MP: 201, PhysAtk: 31, MagicAtk: 102, PhysDef: 33, MagicDef: 79, Spd: 72, Luck: 54}, // プレイヤー2
		{HP: 363, MP: 135, PhysAtk: 81, MagicAtk: 56, PhysDef: 79, MagicDef: 56, Spd: 45, Luck: 27},  // プレイヤー3
		{HP: 297, MP: 102, PhysAtk: 59, MagicAtk: 56, PhysDef: 56, MagicDef: 56, Spd: 89, Luck: 79},  // プレイヤー4
	},
	{ // Lv25
		{HP: 388, MP: 111, PhysAtk: 111, MagicAtk: 34, PhysDef: 58, MagicDef: 34, Spd: 48, Luck: 29}, // プレイヤー1
		{HP: 262, MP: 208, PhysAtk: 32, MagicAtk: 106, PhysDef: 34, MagicDef: 82, Spd: 74, Luck: 56}, // プレイヤー2
		{HP: 374, MP: 140, PhysAtk: 84, MagicAtk: 58, PhysDef: 82, MagicDef: 58, Spd: 46, Luck: 28},  // プレイヤー3
		{HP: 306, MP: 106, PhysAtk: 61, MagicAtk: 58, PhysDef: 58, MagicDef: 58, Spd: 92, Luck: 82},  // プレイヤー4
	},
	{ // Lv26
		{HP: 400, MP: 115, PhysAtk: 115, MagicAtk: 35, PhysDef: 60, MagicDef: 35, Spd: 49, Luck: 30}, // プレイヤー1
		{HP: 270, MP: 215, PhysAtk: 33, MagicAtk: 110, PhysDef: 35, MagicDef: 85, Spd: 76, Luck: 58}, // プレイヤー2
		{HP: 385, MP: 145, PhysAtk: 87, MagicAtk: 60, PhysDef: 85, MagicDef: 60, Spd: 47, Luck: 29},  // プレイヤー3
		{HP: 315, MP: 110, PhysAtk: 63, MagicAtk: 60, PhysDef: 60, MagicDef: 60, Spd: 95, Luck: 85},  // プレイヤー4
	},
	{ // Lv27
		{HP: 412, MP: 119, PhysAtk: 119, MagicAtk: 36, PhysDef: 62, MagicDef: 36, Spd: 50, Luck: 31}, // プレイヤー1
		{HP: 278, MP: 222, PhysAtk: 34, MagicAtk: 114, PhysDef: 36, MagicDef: 88, Spd: 78, Luck: 60}, // プレイヤー2
		{HP: 396, MP: 150, PhysAtk: 90, MagicAtk: 62, PhysDef: 88, MagicDef: 62, Spd: 48, Luck: 30},  // プレイヤー3
		{HP: 324, MP: 114, PhysAtk: 65, MagicAtk: 62, PhysDef: 62, MagicDef: 62, Spd: 98, Luck: 88},  // プレイヤー4
	},
	{ // Lv28
		{HP: 424, MP: 123, PhysAtk: 123, MagicAtk: 37, PhysDef: 64, MagicDef: 37, Spd: 51, Luck: 32}, // プレイヤー1
		{HP: 286, MP: 229, PhysAtk: 35, MagicAtk: 118, PhysDef: 37, MagicDef: 91, Spd: 80, Luck: 62}, // プレイヤー2
		{HP: 407, MP: 155, PhysAtk: 93, MagicAtk: 64, PhysDef: 91, MagicDef: 64, Spd: 49, Luck: 31},  // プレイヤー3
		{HP: 333, MP: 118, PhysAtk: 67, MagicAtk: 64, PhysDef: 64, MagicDef: 64, Spd: 101, Luck: 91}, // プレイヤー4
	},
	{ // Lv29
		{HP: 436, MP: 127, PhysAtk: 127, MagicAtk: 38, PhysDef: 66, MagicDef: 38, Spd: 52, Luck: 33}, // プレイヤー1
		{HP: 294, MP: 236, PhysAtk: 36, MagicAtk: 122, PhysDef: 38, MagicDef: 94, Spd: 82, Luck: 64}, // プレイヤー2
		{HP: 418, MP: 160, PhysAtk: 96, MagicAtk: 66, PhysDef: 94, MagicDef: 66, Spd: 50, Luck: 32},  // プレイヤー3
		{HP: 342, MP: 122, PhysAtk: 69, MagicAtk: 66, PhysDef: 66, MagicDef: 66, Spd: 104, Luck: 94}, // プレイヤー4
	},
	{ // Lv30
		{HP: 448, MP: 131, PhysAtk: 131, MagicAtk: 39, PhysDef: 68, MagicDef: 39, Spd: 53, Luck: 34}, // プレイヤー1
		{HP: 302, MP: 243, PhysAtk: 37, MagicAtk: 126, PhysDef: 39, MagicDef: 97, Spd: 84, Luck: 66}, // プレイヤー2
		{HP: 429, MP: 165, PhysAtk: 99, MagicAtk: 68, PhysDef: 97, MagicDef: 68, Spd: 51, Luck: 33},  // プレイヤー3
		{HP: 351, MP: 126, PhysAtk: 71, MagicAtk: 68, PhysDef: 68, MagicDef: 68, Spd: 107, Luck: 97}, // プレイヤー4
	},
	{ // Lv31
		{HP: 460, MP: 135, PhysAtk: 135, MagicAtk: 40, PhysDef: 70, MagicDef: 40, Spd: 54, Luck: 35},  // プレイヤー1
		{HP: 310, MP: 250, PhysAtk: 38, MagicAtk: 130, PhysDef: 40, MagicDef: 100, Spd: 86, Luck: 68}, // プレイヤー2
		{HP: 440, MP: 170, PhysAtk: 102, MagicAtk: 70, PhysDef: 100, MagicDef: 70, Spd: 52, Luck: 34}, // プレイヤー3
		{HP: 360, MP: 130, PhysAtk: 73, MagicAtk: 70, PhysDef: 70, MagicDef: 70, Spd: 110, Luck: 100}, // プレイヤー4
	},
	{ // Lv32
		{HP: 472, MP: 139, PhysAtk: 139, MagicAtk: 41, PhysDef: 72, MagicDef: 41, Spd: 55, Luck: 36},  // プレイヤー1
		{HP: 318, MP: 257, PhysAtk: 39, MagicAtk: 134, PhysDef: 41, MagicDef: 103, Spd: 88, Luck: 70}, // プレイヤー2
		{HP: 451, MP: 175, PhysAtk: 105, MagicAtk: 72, PhysDef: 103, MagicDef: 72, Spd: 53, Luck: 35}, // プレイヤー3
		{HP: 369, MP: 134, PhysAtk: 75, MagicAtk: 72, PhysDef: 72, MagicDef: 72, Spd: 113, Luck: 103}, // プレイヤー4
	},
	{ // Lv33
		{HP: 484, MP: 143, PhysAtk: 143, MagicAtk: 42, PhysDef: 74, MagicDef: 42, Spd: 56, Luck: 37},  // プレイヤー1
		{HP: 326, MP: 264, PhysAtk: 40, MagicAtk: 138, PhysDef: 42, MagicDef: 106, Spd: 90, Luck: 72}, // プレイヤー2
		{HP: 462, MP: 180, PhysAtk: 108, MagicAtk: 74, PhysDef: 106, MagicDef: 74, Spd: 54, Luck: 36}, // プレイヤー3
		{HP: 378, MP: 138, PhysAtk: 77, MagicAtk: 74, PhysDef: 74, MagicDef: 74, Spd: 116, Luck: 106}, // プレイヤー4
	},
	{ // Lv34
		{HP: 496, MP: 147, PhysAtk: 147, MagicAtk: 43, PhysDef: 76, MagicDef: 43, Spd: 57, Luck: 38},  // プレイヤー1
		{HP: 334, MP: 271, PhysAtk: 41, MagicAtk: 142, PhysDef: 43, MagicDef: 109, Spd: 92, Luck: 74}, // プレイヤー2
		{HP: 473, MP: 185, PhysAtk: 111, MagicAtk: 76, PhysDef: 109, MagicDef: 76, Spd: 55, Luck: 37}, // プレイヤー3
		{HP: 387, MP: 142, PhysAtk: 79, MagicAtk: 76, PhysDef: 76, MagicDef: 76, Spd: 119, Luck: 109}, // プレイヤー4
	},
	{ // Lv35
		{HP: 508, MP: 151, PhysAtk: 151, MagicAtk: 44, PhysDef: 78, MagicDef: 44, Spd: 58, Luck: 39},  // プレイヤー1
		{HP: 342, MP: 278, PhysAtk: 42, MagicAtk: 146, PhysDef: 44, MagicDef: 112, Spd: 94, Luck: 76}, // プレイヤー2
		{HP: 484, MP: 190, PhysAtk: 114, MagicAtk: 78, PhysDef: 112, MagicDef: 78, Spd: 56, Luck: 38}, // プレイヤー3
		{HP: 396, MP: 146, PhysAtk: 81, MagicAtk: 78, PhysDef: 78, MagicDef: 78, Spd: 122, Luck: 112}, // プレイヤー4
	},
	{ // Lv36
		{HP: 520, MP: 155, PhysAtk: 155, MagicAtk: 45, PhysDef: 80, MagicDef: 45, Spd: 59, Luck: 40},  // プレイヤー1
		{HP: 350, MP: 285, PhysAtk: 43, MagicAtk: 150, PhysDef: 45, MagicDef: 115, Spd: 96, Luck: 78}, // プレイヤー2
		{HP: 495, MP: 195, PhysAtk: 117, MagicAtk: 80, PhysDef: 115, MagicDef: 80, Spd: 57, Luck: 39}, // プレイヤー3
		{HP: 405, MP: 150, PhysAtk: 83, MagicAtk: 80, PhysDef: 80, MagicDef: 80, Spd: 125, Luck: 115}, // プレイヤー4
	},
	{ // Lv37
		{HP: 532, MP: 159, PhysAtk: 159, MagicAtk: 46, PhysDef: 82, MagicDef: 46, Spd: 60, Luck: 41},  // プレイヤー1
		{HP: 358, MP: 292, PhysAtk: 44, MagicAtk: 154, PhysDef: 46, MagicDef: 118, Spd: 98, Luck: 80}, // プレイヤー2
		{HP: 506, MP: 200, PhysAtk: 120, MagicAtk: 82, PhysDef: 118, MagicDef: 82, Spd: 58, Luck: 40}, // プレイヤー3
		{HP: 414, MP: 154, PhysAtk: 85, MagicAtk: 82, PhysDef: 82, MagicDef: 82, Spd: 128, Luck: 118}, // プレイヤー4
	},
	{ // Lv38
		{HP: 544, MP: 163, PhysAtk: 163, MagicAtk: 47, PhysDef: 84, MagicDef: 47, Spd: 61, Luck: 42},   // プレイヤー1
		{HP: 366, MP: 299, PhysAtk: 45, MagicAtk: 158, PhysDef: 47, MagicDef: 121, Spd: 100, Luck: 82}, // プレイヤー2
		{HP: 517, MP: 205, PhysAtk: 123, MagicAtk: 84, PhysDef: 121, MagicDef: 84, Spd: 59, Luck: 41},  // プレイヤー3
		{HP: 423, MP: 158, PhysAtk: 87, MagicAtk: 84, PhysDef: 84, MagicDef: 84, Spd: 131, Luck: 121},  // プレイヤー4
	},
	{ // Lv39
		{HP: 556, MP: 167, PhysAtk: 167, MagicAtk: 48, PhysDef: 86, MagicDef: 48, Spd: 62, Luck: 43},   // プレイヤー1
		{HP: 374, MP: 306, PhysAtk: 46, MagicAtk: 162, PhysDef: 48, MagicDef: 124, Spd: 102, Luck: 84}, // プレイヤー2
		{HP: 528, MP: 210, PhysAtk: 126, MagicAtk: 86, PhysDef: 124, MagicDef: 86, Spd: 60, Luck: 42},  // プレイヤー3
		{HP: 432, MP: 162, PhysAtk: 89, MagicAtk: 86, PhysDef: 86, MagicDef: 86, Spd: 134, Luck: 124},  // プレイヤー4
	},
	{ // Lv40
		{HP: 568, MP: 171, PhysAtk: 171, MagicAtk: 49, PhysDef: 88, MagicDef: 49, Spd: 63, Luck: 44},   // プレイヤー1
		{HP: 382, MP: 313, PhysAtk: 47, MagicAtk: 166, PhysDef: 49, MagicDef: 127, Spd: 104, Luck: 86}, // プレイヤー2
		{HP: 539, MP: 215, PhysAtk: 129, MagicAtk: 88, PhysDef: 127, MagicDef: 88, Spd: 61, Luck: 43},  // プレイヤー3
		{HP: 441, MP: 166, PhysAtk: 91, MagicAtk: 88, PhysDef: 88, MagicDef: 88, Spd: 137, Luck: 127},  // プレイヤー4
	},
	{ // Lv41
		{HP: 580, MP: 175, PhysAtk: 175, MagicAtk: 50, PhysDef: 90, MagicDef: 50, Spd: 64, Luck: 45},   // プレイヤー1
		{HP: 390, MP: 320, PhysAtk: 48, MagicAtk: 170, PhysDef: 50, MagicDef: 130, Spd: 106, Luck: 88}, // プレイヤー2
		{HP: 550, MP: 220, PhysAtk: 132, MagicAtk: 90, PhysDef: 130, MagicDef: 90, Spd: 62, Luck: 44},  // プレイヤー3
		{HP: 450, MP: 170, PhysAtk: 93, MagicAtk: 90, PhysDef: 90, MagicDef: 90, Spd: 140, Luck: 130},  // プレイヤー4
	},
	{ // Lv42
		{HP: 592, MP: 179, PhysAtk: 179, MagicAtk: 51, PhysDef: 92, MagicDef: 51, Spd: 65, Luck: 46},   // プレイヤー1
		{HP: 398, MP: 327, PhysAtk: 49, MagicAtk: 174, PhysDef: 51, MagicDef: 133, Spd: 108, Luck: 90}, // プレイヤー2
		{HP: 561, MP: 225, PhysAtk: 135, MagicAtk: 92, PhysDef: 133, MagicDef: 92, Spd: 63, Luck: 45},  // プレイヤー3
		{HP: 459, MP: 174, PhysAtk: 95, MagicAtk: 92, PhysDef: 92, MagicDef: 92, Spd: 143, Luck: 133},  // プレイヤー4
	},
	{ // Lv43
		{HP: 604, MP: 183, PhysAtk: 183, MagicAtk: 52, PhysDef: 94, MagicDef: 52, Spd: 66, Luck: 47},   // プレイヤー1
		{HP: 406, MP: 334, PhysAtk: 50, MagicAtk: 178, PhysDef: 52, MagicDef: 136, Spd: 110, Luck: 92}, // プレイヤー2
		{HP: 572, MP: 230, PhysAtk: 138, MagicAtk: 94, PhysDef: 136, MagicDef: 94, Spd: 64, Luck: 46},  // プレイヤー3
		{HP: 468, MP: 178, PhysAtk: 97, MagicAtk: 94, PhysDef: 94, MagicDef: 94, Spd: 146, Luck: 136},  // プレイヤー4
	},
	{ // Lv44
		{HP: 616, MP: 187, PhysAtk: 187, MagicAtk: 53, PhysDef: 96, MagicDef: 53, Spd: 67, Luck: 48},   // プレイヤー1
		{HP: 414, MP: 341, PhysAtk: 51, MagicAtk: 182, PhysDef: 53, MagicDef: 139, Spd: 112, Luck: 94}, // プレイヤー2
		{HP: 583, MP: 235, PhysAtk: 141, MagicAtk: 96, PhysDef: 139, MagicDef: 96, Spd: 65, Luck: 47},  // プレイヤー3
		{HP: 477, MP: 182, PhysAtk: 99, MagicAtk: 96, PhysDef: 96, MagicDef: 96, Spd: 149, Luck: 139},  // プレイヤー4
	},
	{ // Lv45
		{HP: 628, MP: 191, PhysAtk: 191, MagicAtk: 54, PhysDef: 98, MagicDef: 54, Spd: 68, Luck: 49},   // プレイヤー1
		{HP: 422, MP: 348, PhysAtk: 52, MagicAtk: 186, PhysDef: 54, MagicDef: 142, Spd: 114, Luck: 96}, // プレイヤー2
		{HP: 594, MP: 240, PhysAtk: 144, MagicAtk: 98, PhysDef: 142, MagicDef: 98, Spd: 66, Luck: 48},  // プレイヤー3
		{HP: 486, MP: 186, PhysAtk: 101, MagicAtk: 98, PhysDef: 98, MagicDef: 98, Spd: 152, Luck: 142}, // プレイヤー4
	},
	{ // Lv46
		{HP: 640, MP: 195, PhysAtk: 195, MagicAtk: 55, PhysDef: 100, MagicDef: 55, Spd: 69, Luck: 50},     // プレイヤー1
		{HP: 430, MP: 355, PhysAtk: 53, MagicAtk: 190, PhysDef: 55, MagicDef: 145, Spd: 116, Luck: 98},    // プレイヤー2
		{HP: 605, MP: 245, PhysAtk: 147, MagicAtk: 100, PhysDef: 145, MagicDef: 100, Spd: 67, Luck: 49},   // プレイヤー3
		{HP: 495, MP: 190, PhysAtk: 103, MagicAtk: 100, PhysDef: 100, MagicDef: 100, Spd: 155, Luck: 145}, // プレイヤー4
	},
	{ // Lv47
		{HP: 652, MP: 199, PhysAtk: 199, MagicAtk: 56, PhysDef: 102, MagicDef: 56, Spd: 70, Luck: 51},     // プレイヤー1
		{HP: 438, MP: 362, PhysAtk: 54, MagicAtk: 194, PhysDef: 56, MagicDef: 148, Spd: 118, Luck: 100},   // プレイヤー2
		{HP: 616, MP: 250, PhysAtk: 150, MagicAtk: 102, PhysDef: 148, MagicDef: 102, Spd: 68, Luck: 50},   // プレイヤー3
		{HP: 504, MP: 194, PhysAtk: 105, MagicAtk: 102, PhysDef: 102, MagicDef: 102, Spd: 158, Luck: 148}, // プレイヤー4
	},
	{ // Lv48
		{HP: 664, MP: 203, PhysAtk: 203, MagicAtk: 57, PhysDef: 104, MagicDef: 57, Spd: 71, Luck: 52},     // プレイヤー1
		{HP: 446, MP: 369, PhysAtk: 55, MagicAtk: 198, PhysDef: 57, MagicDef: 151, Spd: 120, Luck: 102},   // プレイヤー2
		{HP: 627, MP: 255, PhysAtk: 153, MagicAtk: 104, PhysDef: 151, MagicDef: 104, Spd: 69, Luck: 51},   // プレイヤー3
		{HP: 513, MP: 198, PhysAtk: 107, MagicAtk: 104, PhysDef: 104, MagicDef: 104, Spd: 161, Luck: 151}, // プレイヤー4
	},
	{ // Lv49
		{HP: 676, MP: 207, PhysAtk: 207, MagicAtk: 58, PhysDef: 106, MagicDef: 58, Spd: 72, Luck: 53},     // プレイヤー1
		{HP: 454, MP: 376, PhysAtk: 56, MagicAtk: 202, PhysDef: 58, MagicDef: 154, Spd: 122, Luck: 104},   // プレイヤー2
		{HP: 638, MP: 260, PhysAtk: 156, MagicAtk: 106, PhysDef: 154, MagicDef: 106, Spd: 70, Luck: 52},   // プレイヤー3
		{HP: 522, MP: 202, PhysAtk: 109, MagicAtk: 106, PhysDef: 106, MagicDef: 106, Spd: 164, Luck: 154}, // プレイヤー4
	},
	{ // Lv50
		{HP: 688, MP: 211, PhysAtk: 211, MagicAtk: 59, PhysDef: 108, MagicDef: 59, Spd: 73, Luck: 54},     // プレイヤー1
		{HP: 462, MP: 383, PhysAtk: 57, MagicAtk: 206, PhysDef: 59, MagicDef: 157, Spd: 124, Luck: 106},   // プレイヤー2
		{HP: 649, MP: 265, PhysAtk: 159, MagicAtk: 108, PhysDef: 157, MagicDef: 108, Spd: 71, Luck: 53},   // プレイヤー3
		{HP: 531, MP: 206, PhysAtk: 111, MagicAtk: 108, PhysDef: 108, MagicDef: 108, Spd: 167, Luck: 157}, // プレイヤー4
	},
}

// PlayerExpToNextByLevel[レベル-1]：そのレベルから次のレベルに必要なEXP。
// EXPは全キャラ共通で、レベルごとに1つだけ設定する。
var PlayerExpToNextByLevel = [maxPlayerLevel]int{
	65,   // Lv1
	110,  // Lv2
	155,  // Lv3
	200,  // Lv4
	245,  // Lv5
	290,  // Lv6
	335,  // Lv7
	380,  // Lv8
	425,  // Lv9
	470,  // Lv10
	515,  // Lv11
	560,  // Lv12
	605,  // Lv13
	650,  // Lv14
	695,  // Lv15
	740,  // Lv16
	785,  // Lv17
	830,  // Lv18
	875,  // Lv19
	920,  // Lv20
	965,  // Lv21
	1010, // Lv22
	1055, // Lv23
	1100, // Lv24
	1145, // Lv25
	1190, // Lv26
	1235, // Lv27
	1280, // Lv28
	1325, // Lv29
	1370, // Lv30
	1415, // Lv31
	1460, // Lv32
	1505, // Lv33
	1550, // Lv34
	1595, // Lv35
	1640, // Lv36
	1685, // Lv37
	1730, // Lv38
	1775, // Lv39
	1820, // Lv40
	1865, // Lv41
	1910, // Lv42
	1955, // Lv43
	2000, // Lv44
	2045, // Lv45
	2090, // Lv46
	2135, // Lv47
	2180, // Lv48
	2225, // Lv49
	2270, // Lv50
}

// ── 敵ステータス ────────────────────────────────────────

// EnemyStats は敵1体分のステータス一式。味方と違い「運」は持たない。
type EnemyStats struct {
	Name          string
	Lv            int // レベル
	Exp           int // 撃破時に味方へ与えるEXP
	HP            int // 最大HP
	MP            int // 最大MP（現状は演出/将来のスキル用の保持値）
	PhysAtk       int // 物理攻撃力
	MagicAtk      int // 魔法攻撃力
	PhysDef       int // 物理防御力
	MagicDef      int // 魔法防御力
	Spd           int // すばやさ（タイムライン上でアイコンが進む速さ）
	SP            int // 撃破時に味方へ与えるSP（ステータスではなく戦闘報酬）
	Element       Element
	ElementResist [elementalTypeCount]int // 火・雷・氷・風の耐性（%）。負数は弱点。
	Drops         []ItemDrop              // 撃破時のアイテムドロップ抽選テーブル（自由に追加・変更可。MinCount/MaxCountを指定すると個数に幅が出る。省略時は1個）
}

// EnemyDatabase：通常敵。ここを直接編集すればステータスを調整できる。
var EnemyDatabase = []EnemyStats{
	{Name: "フリーザ", Lv: -10, Exp: 45, HP: 160, MP: 20, PhysAtk: 18, MagicAtk: 14, PhysDef: 8, MagicDef: 8, Spd: 22, SP: 2, Element: ElemIce, ElementResist: [4]int{-30, 0, 30, 0},
		Drops: []ItemDrop{{ItemID: "potion", Percent: 100}}},
	{Name: "セル", Lv: -10, Exp: 60, HP: 200, MP: 30, PhysAtk: 22, MagicAtk: 18, PhysDef: 10, MagicDef: 9, Spd: 26, SP: 3, Element: ElemWind, ElementResist: [4]int{0, 0, -30, 30},
		Drops: []ItemDrop{{ItemID: "hi_potion", Percent: 100}, {ItemID: "ether", Percent: 20}}},
	{Name: "魔人ブウ", Lv: -10, Exp: 90, HP: 350, MP: 40, PhysAtk: 16, MagicAtk: 20, PhysDef: 12, MagicDef: 14, Spd: 16, SP: 4, Element: ElemLightning, ElementResist: [4]int{0, 30, 0, -30},
		Drops: []ItemDrop{{ItemID: "ether", Percent: 100}, {ItemID: "phoenix_down", Percent: 10}}},
}

// BossDatabase：ボス敵。ここを直接編集すればステータスを調整できる。
var BossDatabase = map[string]EnemyStats{
	"boss_1": {Name: BossNames[0], Lv: 15, Exp: 80, HP: 200, MP: 50, PhysAtk: 20, MagicAtk: 16, PhysDef: 12, MagicDef: 10, Spd: 18, SP: 300, Element: ElemFire, ElementResist: [4]int{30, -30, 0, 0},
		Drops: []ItemDrop{{ItemID: "hi_potion", Percent: 100}}},
	"boss_2": {Name: BossNames[1], Lv: 20, Exp: 100, HP: 250, MP: 60, PhysAtk: 24, MagicAtk: 19, PhysDef: 14, MagicDef: 12, Spd: 20, SP: 400, Element: ElemLightning, ElementResist: [4]int{0, 30, -30, 0},
		Drops: []ItemDrop{{ItemID: "ether", Percent: 100}}},
	"boss_3": {Name: BossNames[2], Lv: 25, Exp: 130, HP: 300, MP: 70, PhysAtk: 27, MagicAtk: 22, PhysDef: 16, MagicDef: 14, Spd: 22, SP: 500, Element: ElemIce, ElementResist: [4]int{0, -30, 30, 0},
		Drops: []ItemDrop{{ItemID: "phoenix_down", Percent: 100}}},
	"boss_4": {Name: BossNames[3], Lv: 35, Exp: 500, HP: 600, MP: 100, PhysAtk: 40, MagicAtk: 32, PhysDef: 26, MagicDef: 24, Spd: 30, SP: 1919, Element: ElemWind, ElementResist: [4]int{0, 0, -30, 30},
		Drops: []ItemDrop{{ItemID: "elixir", Percent: 100}}},
}
