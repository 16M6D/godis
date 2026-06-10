# godis

`godis` 是一个类 Redis 项目，用于实现一个轻量级的内存键值数据库，并尝试复现 Redis 的部分核心能力与设计思路。

这个项目适合作为：

- 学习 Redis 基本原理的练手项目
- 理解高性能缓存系统设计的实验项目
- 练习网络编程、数据结构与并发处理的实践项目

## Quick Start
```shell
go build . 
./godis # 默认使用conf.json
./godis conf.json # 指定config路径

# 支持redis-cli使用
redis-cli -p 6657
```


## 项目定位

这是一个 **类 Redis（Redis-like）** 项目，而不是 Redis 的完整实现。  
项目目标是围绕 Redis 的核心思想，逐步实现一个简化版本的服务端，包括但不限于：

- 基于内存的 Key-Value 数据存储
- 常见数据类型支持
- 基本命令解析与执行
- TCP 服务与客户端连接处理
- 过期键管理
- 持久化或其他扩展能力
- 主从复制
- cluster模式

## 项目说明

当前仓库用于开发一个类 Redis 系统，重点关注：

- 简洁的架构设计
- 清晰的模块划分
- 易于理解和扩展的实现方式

随着开发推进，项目结构、支持命令和运行方式会逐步补充。

本项目是一个学习性 / 实验性项目，不以完全兼容官方 Redis 为目标。  
如果你需要生产级 Redis 功能，请使用官方 Redis：

[https://redis.io/](https://redis.io/)
