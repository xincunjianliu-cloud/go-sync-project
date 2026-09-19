package main

type PlayerStats struct {
	HP       int
	MP       int
	PhysAtk  int
	MagicAtk int
	PhysDef  int
	MagicDef int
	Spd      int
	Luck     int
}

type EnemyStats struct {
	Name          string
	Lv            int
	Exp           int
	HP            int
	MP            int
	PhysAtk       int
	MagicAtk      int
	PhysDef       int
	MagicDef      int
	Spd           int
	SP            int
	Element       Element
	ElementResist [elementalTypeCount]int
	Drops         []ItemDrop
	Skills        []EnemySkill
}
