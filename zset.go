package main

import (
	"math/rand"
)

const (
	ZSKIPLIST_MAXLEVEL int = 32   // 最大层数
	ZSKIPLIST_P        int = 0x40 // p = 1/4, Redis 默认值
)

// skiplist node
type ZSkipListNode struct {
	member   *Gobj
	score    float64
	backward *ZSkipListNode
	levels   []ZSkipListLevel
}

type ZSkipListLevel struct {
	forward *ZSkipListNode
	span    int64 // 跨度, 用于 rank 计算
}

type ZSkipList struct {
	head   *ZSkipListNode
	tail   *ZSkipListNode
	length int64
	level  int // 当前最大层数 (1-32)
}

// ---- skiplist create / free ----

func ZSkipListCreate() *ZSkipList {
	zsl := &ZSkipList{
		level: 1,
	}
	// head 节点不存数据, 但有 maxLevel 层
	zsl.head = &ZSkipListNode{
		levels: make([]ZSkipListLevel, ZSKIPLIST_MAXLEVEL),
	}
	return zsl
}

func (zsl *ZSkipList) free() {
	n := zsl.head.levels[0].forward
	for n != nil {
		next := n.levels[0].forward
		n.member.DecrRefCount()
		n = next
	}
}

// 随机生成层数 (1 ~ ZSKIPLIST_MAXLEVEL)
func zslRandomLevel() int {
	level := 1
	// rand & 0xFFFF < 0x4000 → P = 1/4
	const threshold = ZSKIPLIST_P * 0x100
	for (rand.Int() & 0xFFFF) < threshold {
		level++
	}
	if level > ZSKIPLIST_MAXLEVEL {
		level = ZSKIPLIST_MAXLEVEL
	}
	return level
}

// ---- skiplist insert ----

// 插入一个新节点, score + member. 返回插入的节点
func (zsl *ZSkipList) insert(score float64, member *Gobj) *ZSkipListNode {
	// update[i] 记录每层需要更新的前驱节点
	var update [ZSKIPLIST_MAXLEVEL]*ZSkipListNode
	// rank[i] 记录每层 update[i] 到 head 的跨度
	var rank [ZSKIPLIST_MAXLEVEL]int64

	x := zsl.head
	for i := zsl.level - 1; i >= 0; i-- {
		if i == zsl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		// 在同一层中前进, 找到插入位置:
		// forward.score < score 或 (score 相等但 member 字典序更小)
		for x.levels[i].forward != nil &&
			(x.levels[i].forward.score < score ||
				(x.levels[i].forward.score == score &&
					x.levels[i].forward.member.StrVal() < member.StrVal())) {
			rank[i] += x.levels[i].span
			x = x.levels[i].forward
		}
		update[i] = x
	}

	// 生成新节点层数
	level := zslRandomLevel()
	if level > zsl.level {
		for i := zsl.level; i < level; i++ {
			rank[i] = 0
			update[i] = zsl.head
			update[i].levels[i].span = zsl.length
		}
		zsl.level = level
	}

	// 创建节点
	node := &ZSkipListNode{
		member: member,
		score:  score,
		levels: make([]ZSkipListLevel, level),
	}
	member.IncrRefCount()

	// 在每层插入
	for i := 0; i < level; i++ {
		node.levels[i].forward = update[i].levels[i].forward
		update[i].levels[i].forward = node
		// 更新 span
		node.levels[i].span = update[i].levels[i].span - (rank[0] - rank[i])
		update[i].levels[i].span = (rank[0] - rank[i]) + 1
	}

	// 对 level 以上的层, 前驱的 span +1 (新节点在它们的"下方")
	for i := level; i < zsl.level; i++ {
		update[i].levels[i].span++
	}

	// 设置 backward 指针
	if update[0] == zsl.head {
		node.backward = nil
	} else {
		node.backward = update[0]
	}
	if node.levels[0].forward != nil {
		node.levels[0].forward.backward = node
	} else {
		zsl.tail = node
	}

	zsl.length++
	return node
}

// ---- skiplist delete ----

// 内部删除: 给定 node 和 update 数组, 从 skiplist 中移除
func (zsl *ZSkipList) deleteNode(node *ZSkipListNode, update [ZSKIPLIST_MAXLEVEL]*ZSkipListNode) {
	for i := 0; i < zsl.level; i++ {
		if update[i].levels[i].forward == node {
			update[i].levels[i].span += node.levels[i].span - 1
			update[i].levels[i].forward = node.levels[i].forward
		} else {
			update[i].levels[i].span -= 1
		}
	}
	// 更新 backward
	if node.levels[0].forward != nil {
		node.levels[0].forward.backward = node.backward
	} else {
		zsl.tail = node.backward
	}
	// 收缩 level
	for zsl.level > 1 && zsl.head.levels[zsl.level-1].forward == nil {
		zsl.level--
	}
	zsl.length--
}

// 根据 score + member 删除节点, 返回 1 成功 / 0 没找到
func (zsl *ZSkipList) delete(score float64, member *Gobj) int {
	var update [ZSKIPLIST_MAXLEVEL]*ZSkipListNode
	x := zsl.head
	for i := zsl.level - 1; i >= 0; i-- {
		for x.levels[i].forward != nil &&
			(x.levels[i].forward.score < score ||
				(x.levels[i].forward.score == score &&
					x.levels[i].forward.member.StrVal() < member.StrVal())) {
			x = x.levels[i].forward
		}
		update[i] = x
	}
	x = x.levels[0].forward
	if x != nil && x.score == score && x.member.StrVal() == member.StrVal() {
		zsl.deleteNode(x, update)
		x.member.DecrRefCount()
		return 1
	}
	return 0
}

// ---- skiplist search ----

// 获取 member 的 rank (0-based), 从 0 开始
func (zsl *ZSkipList) getRank(score float64, member *Gobj) int64 {
	var rank int64 = 0
	x := zsl.head
	for i := zsl.level - 1; i >= 0; i-- {
		for x.levels[i].forward != nil &&
			(x.levels[i].forward.score < score ||
				(x.levels[i].forward.score == score &&
					x.levels[i].forward.member.StrVal() <= member.StrVal())) {
			rank += x.levels[i].span
			x = x.levels[i].forward
		}
		if x.member != nil && x.member.StrVal() == member.StrVal() {
			return rank - 1
		}
	}
	return -1
}

// getElementByRank 根据 rank (1-based, 与 Redis 一致) 获取节点.
// rank=1 返回第一个元素.
func (zsl *ZSkipList) getElementByRank(rank int64) *ZSkipListNode {
	if rank <= 0 || rank > zsl.length {
		return nil
	}
	var traversed int64 = 0
	x := zsl.head
	for i := zsl.level - 1; i >= 0; i-- {
		for x.levels[i].forward != nil && traversed+x.levels[i].span <= rank {
			traversed += x.levels[i].span
			x = x.levels[i].forward
		}
		// Redis 注释: "The rank argument needs to be 1-based."
		// 当 traversed == rank 时, x 就是目标节点 (因为 header span 计数保证了这一点)
		if traversed == rank && x != zsl.head {
			return x
		}
	}
	return nil
}

// ---- ZSet: skiplist + dict ----

type ZSet struct {
	sl   *ZSkipList
	dict *Dict // member → score (*Gobj, float64 stored as string)
}

func ZSetCreate(dictType DictType) *ZSet {
	return &ZSet{
		sl:   ZSkipListCreate(),
		dict: DictCreate(dictType),
	}
}

func (zs *ZSet) free() {
	zs.sl.free()
	// dict 和 entry 的 GC 由 Go 处理
}
