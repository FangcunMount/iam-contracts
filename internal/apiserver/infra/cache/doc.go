// Package cache 定义 IAM 缓存层的静态模型与治理目录。
//
// 这一层只负责回答三个问题：
// 1. 当前有哪些 cache family；
// 2. 每个 family 使用什么后端、什么 Redis 数据结构、什么编码；
// 3. 第一版治理面可以如何只读地观察它们。
//
// 它不负责实际 Redis 读写，也不负责控制动作。
package cache
