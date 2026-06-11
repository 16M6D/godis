package main

import (
	"fmt"
	"strconv"
)

// ---- helpers ----

func expireIfNeeded(key *Gobj) {
	entry := server.db.expire.Find(key)
	if entry == nil {
		return
	}
	when := entry.Val.IntVal()
	if when > GetMsTime() {
		return
	}
	server.db.expire.Delete(key)
	server.db.data.Delete(key)
}

func findKeyRead(key *Gobj) *Gobj {
	expireIfNeeded(key)
	return server.db.data.Get(key)
}

func replyWrongType(c *GodisClient) {
	c.AddReplyStr("-ERR: wrong type\r\n")
}

// ---- string commands ----

func getCommand(c *GodisClient) {
	key := c.args[1]
	val := findKeyRead(key)
	if val == nil {
		c.AddReplyStr("$-1\r\n")
	} else if val.Type_ != GSTR {
		replyWrongType(c)
	} else {
		str := val.StrVal()
		c.AddReplyStr(fmt.Sprintf("$%d\r\n%v\r\n", len(str), str))
	}
}

func setCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != GSTR {
		replyWrongType(c)
		return
	}
	server.db.data.Set(key, val)
	server.db.expire.Delete(key)
	c.AddReplyStr("+OK\r\n")
}

func expireCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != GSTR {
		replyWrongType(c)
		return
	}
	expire := GetMsTime() + (val.IntVal() * 1000)
	expObj := CreateFromInt(expire)
	server.db.expire.Set(key, expObj)
	expObj.DecrRefCount()
	c.AddReplyStr("+OK\r\n")
}

func delCommand(c *GodisClient) {
	deleted := 0
	for i := 1; i < len(c.args); i++ {
		if server.db.data.Delete(c.args[i]) == nil {
			deleted++
		}
		server.db.expire.Delete(c.args[i])
	}
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", deleted))
}

func existsCommand(c *GodisClient) {
	count := 0
	for i := 1; i < len(c.args); i++ {
		expireIfNeeded(c.args[i])
		if server.db.data.Find(c.args[i]) != nil {
			count++
		}
	}
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", count))
}

func incrCommand(c *GodisClient) {
	key := c.args[1]
	val := findKeyRead(key)
	if val == nil {
		// key doesn't exist, set to 1
		server.db.data.Set(key, CreateFromInt(1))
		c.AddReplyStr(":1\r\n")
		return
	}
	if val.Type_ != GSTR {
		replyWrongType(c)
		return
	}
	i, err := strconv.ParseInt(val.StrVal(), 10, 64)
	if err != nil {
		c.AddReplyStr("-ERR: value is not an integer\r\n")
		return
	}
	i++
	server.db.data.Set(key, CreateFromInt(i))
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", i))
}

func decrCommand(c *GodisClient) {
	key := c.args[1]
	val := findKeyRead(key)
	if val == nil {
		server.db.data.Set(key, CreateFromInt(-1))
		c.AddReplyStr(":-1\r\n")
		return
	}
	if val.Type_ != GSTR {
		replyWrongType(c)
		return
	}
	i, err := strconv.ParseInt(val.StrVal(), 10, 64)
	if err != nil {
		c.AddReplyStr("-ERR: value is not an integer\r\n")
		return
	}
	i--
	server.db.data.Set(key, CreateFromInt(i))
	c.AddReplyStr(fmt.Sprintf(":%d\r\n", i))
}
