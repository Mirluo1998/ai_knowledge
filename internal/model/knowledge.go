// Package model 定义知识服务的领域模型。
package model

import "time"

// Knowledge 表示一条知识条目。
type Knowledge struct {
	ID        int64     `json:"id"`
	Type      int32     `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// KnowledgeQuery 表示列表查询的过滤条件。
// 与领域模型分离，避免把"数据载体"和"查询参数"混为一个结构。
type KnowledgeQuery struct {
	Type  int32
	Title string
}
