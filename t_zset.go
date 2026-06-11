package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ---- helpers ----

// 将 score (float64) 格式化为字符串, 存到 dict 中
func formatScore(score float64) *Gobj {
	return CreateObject(GSTR, strconv.FormatFloat(score, 'f', -1, 64))
}

// 从 dict 存储的值中解析 score
func parseScore(val *Gobj) (float64, error) {
	if val.Type_ != GSTR {
		return 0, fmt.Errorf("wrong type for score")
	}
	return strconv.ParseFloat(val.StrVal(), 64)
}

// 将 score 范围字符串解析为 (min, max, minExclusive, maxExclusive)
func parseScoreRange(minStr, maxStr string) (min, max float64, minEx, maxEx bool, err error) {
	minEx = strings.HasPrefix(minStr, "(")
	maxEx = strings.HasPrefix(maxStr, "(")

	minStr = strings.TrimPrefix(minStr, "(")
	maxStr = strings.TrimPrefix(maxStr, "(")

	if strings.EqualFold(minStr, "-inf") {
		min = math.Inf(-1)
	} else {
		min, err = strconv.ParseFloat(minStr, 64)
		if err != nil {
			return
		}
	}

	if strings.EqualFold(maxStr, "+inf") {
		max = math.Inf(1)
	} else {
		max, err = strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return
		}
	}
	return
}

// 判断 score 是否在范围内
func scoreInRange(score, min, max float64, minEx, maxEx bool) bool {
	if minEx {
		if score <= min {
			return false
		}
	} else {
		if score < min {
			return false
		}
	}
	if maxEx {
		if score >= max {
			return false
		}
	} else {
		if score > max {
			return false
		}
	}
	return true
}

// 查找 ZSet 类型的 key, 处理过期和类型检查
func lookupZSet(key *Gobj) *ZSet {
	expireIfNeeded(key)
	val := server.db.data.Get(key)
	if val == nil || val.Type_ != GZSET {
		return nil
	}
	return val.Val_.(*ZSet)
}

// ---- ZADD key score member [score member ...] ----

func zaddGenericCommand(c *GodisClient, incr bool) {
	key := c.args[1]
	zs := lookupZSet(key)
	if zs == nil {
		// 创建新 ZSet
		zs = ZSetCreate(DictType{HashFunc: GStrHash, EqualFunc: GStrEqual})
		keyObj := CreateObject(GZSET, zs)
		server.db.data.Set(key, keyObj)
		keyObj.DecrRefCount()
		server.db.expire.Delete(key)
	}

	added := 0
	// 从 args[2] 开始, 每两个一组 (score, member)
	for i := 2; i < len(c.args); i += 2 {
		scoreStr := c.args[i].StrVal()
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			c.AddReplyStr("-ERR: value is not a valid float\r\n")
			return
		}

		member := c.args[i+1]

		// 查找 member 是否已存在
		de := zs.dict.Find(member)
		if de != nil {
			// member 已存在
			curScore, _ := parseScore(de.Val)
			if incr {
				score = curScore + score
			}
			if curScore != score {
				// 分数变了, 需要从 skiplist 中删除再插入
				zs.sl.delete(curScore, member)
				newNode := zs.sl.insert(score, member)
				de.Val = formatScore(score)
				_ = newNode
			}
		} else {
			// 新 member
			node := zs.sl.insert(score, member)
			_ = node
			// 加入 dict
			zs.dict.Add(member, formatScore(score))
			added++
		}
	}

	if incr {
		// ZINCRBY 返回新 score 的字符串表示
		// 对于 ZINCRBY, 只有一个 score-member 对
		if len(c.args) == 4 {
			member := c.args[3]
			de := zs.dict.Find(member)
			if de != nil {
				val, _ := parseScore(de.Val)
				c.AddReplyStr(fmt.Sprintf("$%d\r\n%s\r\n",
					len(strconv.FormatFloat(val, 'f', -1, 64)),
					strconv.FormatFloat(val, 'f', -1, 64)))
			}
		}
	} else {
		c.AddReplyStr(fmt.Sprintf(":%d\r\n", added))
	}
}

func zaddCommand(c *GodisClient) {
	zaddGenericCommand(c, false)
}

// ---- ZCARD key ----

func zcardCommand(c *GodisClient) {
	key := c.args[1]
	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr(":0\r\n")
		return
	}
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", zs.sl.length))
}

// ---- ZSCORE key member ----

func zscoreCommand(c *GodisClient) {
	key := c.args[1]
	member := c.args[2]
	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr("$-1\r\n")
		return
	}
	de := zs.dict.Find(member)
	if de == nil {
		c.AddReplyStr("$-1\r\n")
		return
	}
	score, _ := parseScore(de.Val)
	scoreStr := strconv.FormatFloat(score, 'f', -1, 64)
	c.AddReplyStr(fmt.Sprintf("$%d\r\n%s\r\n", len(scoreStr), scoreStr))
}

// ---- ZRANK key member ----

func zrankGenericCommand(c *GodisClient, reverse bool) {
	key := c.args[1]
	member := c.args[2]
	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr("$-1\r\n")
		return
	}
	de := zs.dict.Find(member)
	if de == nil {
		c.AddReplyStr("$-1\r\n")
		return
	}
	score, _ := parseScore(de.Val)
	rank := zs.sl.getRank(score, member)
	if reverse {
		rank = zs.sl.length - 1 - rank
	}
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", rank))
}

func zrankCommand(c *GodisClient) {
	zrankGenericCommand(c, false)
}

func zrevrankCommand(c *GodisClient) {
	zrankGenericCommand(c, true)
}

// ---- ZRANGE key start stop [WITHSCORES] ----

func zrangeGenericCommand(c *GodisClient, reverse bool) {
	key := c.args[1]
	startStr := c.args[2].StrVal()
	stopStr := c.args[3].StrVal()

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.AddReplyStr("-ERR: value is not an integer\r\n")
		return
	}
	stop, err := strconv.ParseInt(stopStr, 10, 64)
	if err != nil {
		c.AddReplyStr("-ERR: value is not an integer\r\n")
		return
	}

	withScores := false
	if len(c.args) == 5 && strings.EqualFold(c.args[4].StrVal(), "withscores") {
		withScores = true
	}

	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr("*0\r\n")
		return
	}

	llen := zs.sl.length

	// 处理负数索引
	if start < 0 {
		start += llen
	}
	if stop < 0 {
		stop += llen
	}
	if start < 0 {
		start = 0
	}
	if stop >= llen {
		stop = llen - 1
	}
	if start > stop || start >= llen {
		c.AddReplyStr("*0\r\n")
		return
	}

	// 收集结果
	rnge := stop - start + 1
	var result []string

	var node *ZSkipListNode
	if reverse {
		// 从尾部向前
		node = zs.sl.getElementByRank(llen - start)
		for i := int64(0); i < rnge && node != nil; i++ {
			result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(node.member.StrVal()), node.member.StrVal()))
			if withScores {
				scoreStr := strconv.FormatFloat(node.score, 'f', -1, 64)
				result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(scoreStr), scoreStr))
			}
			node = node.backward
		}
	} else {
		node = zs.sl.getElementByRank(start + 1)
		for i := int64(0); i < rnge && node != nil; i++ {
			result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(node.member.StrVal()), node.member.StrVal()))
			if withScores {
				scoreStr := strconv.FormatFloat(node.score, 'f', -1, 64)
				result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(scoreStr), scoreStr))
			}
			node = node.levels[0].forward
		}
	}

	// 构造 RESP Array 响应
	reply := fmt.Sprintf("*%d\r\n", len(result))
	for _, s := range result {
		reply += s
	}
	c.AddReplyStr(reply)
}

func zrangeCommand(c *GodisClient) {
	zrangeGenericCommand(c, false)
}

func zrevrangeCommand(c *GodisClient) {
	zrangeGenericCommand(c, true)
}

// ---- ZRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count] ----

func zrangebyscoreCommand(c *GodisClient) {
	key := c.args[1]
	minStr := c.args[2].StrVal()
	maxStr := c.args[3].StrVal()

	min, max, minEx, maxEx, err := parseScoreRange(minStr, maxStr)
	if err != nil {
		c.AddReplyStr("-ERR: min or max is not a float\r\n")
		return
	}

	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr("*0\r\n")
		return
	}

	withScores := false
	offset := int64(0)
	count := int64(-1) // -1 means no limit

	// 解析剩余参数: [WITHSCORES] [LIMIT offset count]
	idx := 4
	for idx < len(c.args) {
		arg := c.args[idx].StrVal()
		if strings.EqualFold(arg, "withscores") {
			withScores = true
			idx++
		} else if strings.EqualFold(arg, "limit") {
			if idx+2 >= len(c.args) {
				c.AddReplyStr("-ERR: syntax error\r\n")
				return
			}
			offset, err = strconv.ParseInt(c.args[idx+1].StrVal(), 10, 64)
			if err != nil {
				c.AddReplyStr("-ERR: offset is not an integer\r\n")
				return
			}
			count, err = strconv.ParseInt(c.args[idx+2].StrVal(), 10, 64)
			if err != nil {
				c.AddReplyStr("-ERR: count is not an integer\r\n")
				return
			}
			idx += 3
		} else {
			c.AddReplyStr("-ERR: syntax error\r\n")
			return
		}
	}

	// 从 skiplist 中遍历
	// 找到第一个符合条件的节点
	var node *ZSkipListNode
	// 从头开始找第一个 score >= min (或 > min)
	x := zs.sl.head
	for i := zs.sl.level - 1; i >= 0; i-- {
		for x.levels[i].forward != nil {
			s := x.levels[i].forward.score
			if s < min || (s == min && minEx) {
				x = x.levels[i].forward
			} else {
				break
			}
		}
	}
	node = x.levels[0].forward

	// 收集结果, 应用 LIMIT
	var result []string
	var skipped int64 = 0
	var taken int64 = 0

	for node != nil && scoreInRange(node.score, min, max, minEx, maxEx) {
		if skipped < offset {
			skipped++
			node = node.levels[0].forward
			continue
		}
		if count >= 0 && taken >= count {
			break
		}
		result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(node.member.StrVal()), node.member.StrVal()))
		taken++
		if withScores {
			scoreStr := strconv.FormatFloat(node.score, 'f', -1, 64)
			result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(scoreStr), scoreStr))
		}
		node = node.levels[0].forward
	}

	// 构造 RESP Array 响应
	reply := fmt.Sprintf("*%d\r\n", len(result))
	for _, s := range result {
		reply += s
	}
	c.AddReplyStr(reply)
}

// ---- ZREVRANGEBYSCORE key max min [WITHSCORES] [LIMIT offset count] ----

func zrevrangebyscoreCommand(c *GodisClient) {
	key := c.args[1]
	maxStr := c.args[2].StrVal()
	minStr := c.args[3].StrVal()

	// 注意: ZREVRANGEBYSCORE 的参数顺序是 max 在前, min 在后
	min, max, minEx, maxEx, err := parseScoreRange(minStr, maxStr)
	if err != nil {
		c.AddReplyStr("-ERR: min or max is not a float\r\n")
		return
	}

	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr("*0\r\n")
		return
	}

	withScores := false
	offset := int64(0)
	count := int64(-1)

	idx := 4
	for idx < len(c.args) {
		arg := c.args[idx].StrVal()
		if strings.EqualFold(arg, "withscores") {
			withScores = true
			idx++
		} else if strings.EqualFold(arg, "limit") {
			if idx+2 >= len(c.args) {
				c.AddReplyStr("-ERR: syntax error\r\n")
				return
			}
			offset, err = strconv.ParseInt(c.args[idx+1].StrVal(), 10, 64)
			if err != nil {
				c.AddReplyStr("-ERR: offset is not an integer\r\n")
				return
			}
			count, err = strconv.ParseInt(c.args[idx+2].StrVal(), 10, 64)
			if err != nil {
				c.AddReplyStr("-ERR: count is not an integer\r\n")
				return
			}
			idx += 3
		} else {
			c.AddReplyStr("-ERR: syntax error\r\n")
			return
		}
	}

	// 从尾部开始找第一个 score <= max 的节点
	var node *ZSkipListNode
	// 找最后一个 score <= max (或 < max if exclusive)的节点
	x := zs.sl.tail
	for x != nil {
		s := x.score
		if maxEx {
			if s >= max {
				x = x.backward
			} else {
				break
			}
		} else {
			if s > max {
				x = x.backward
			} else {
				break
			}
		}
	}
	node = x

	// 从找到的位置向前 (向 score 小的方向) 遍历
	var result []string
	var skipped int64 = 0
	var taken int64 = 0

	for node != nil && scoreInRange(node.score, min, max, minEx, maxEx) {
		if skipped < offset {
			skipped++
			node = node.backward
			continue
		}
		if count >= 0 && taken >= count {
			break
		}
		result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(node.member.StrVal()), node.member.StrVal()))
		taken++
		if withScores {
			scoreStr := strconv.FormatFloat(node.score, 'f', -1, 64)
			result = append(result, fmt.Sprintf("$%d\r\n%s\r\n", len(scoreStr), scoreStr))
		}
		node = node.backward
	}

	reply := fmt.Sprintf("*%d\r\n", len(result))
	for _, s := range result {
		reply += s
	}
	c.AddReplyStr(reply)
}

// ---- ZREM key member [member ...] ----

func zremCommand(c *GodisClient) {
	key := c.args[1]
	zs := lookupZSet(key)
	if zs == nil {
		c.AddReplyStr(":0\r\n")
		return
	}

	deleted := 0
	for i := 2; i < len(c.args); i++ {
		member := c.args[i]
		de := zs.dict.Find(member)
		if de == nil {
			continue
		}
		score, _ := parseScore(de.Val)
		if zs.sl.delete(score, member) != 0 {
			zs.dict.Delete(member)
			deleted++
		}
	}

	// 如果 ZSet 为空, 删除 key
	if zs.sl.length == 0 {
		server.db.data.Delete(key)
		server.db.expire.Delete(key)
	}

	c.AddReplyStr(fmt.Sprintf(":%d\r\n", deleted))
}

// ---- ZINCRBY key increment member ----

func zincrbyCommand(c *GodisClient) {
	zaddGenericCommand(c, true)
}
