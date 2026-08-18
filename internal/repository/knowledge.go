// Package repository 实现知识服务的数据访问层。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"knowledge/internal/model"
)

// knowledgeRepository 是知识条目数据访问的 MySQL 实现。
// 接口由消费方（service 层）定义，这里只返回具体实现，
// 遵循 Go 的 “accept interfaces, return structs” 惯例。
type knowledgeRepository struct {
	db *sql.DB
}

// NewKnowledgeRepository 创建知识条目仓储实例。
func NewKnowledgeRepository(db *sql.DB) *knowledgeRepository {
	return &knowledgeRepository{db: db}
}

// List 根据查询条件返回知识条目列表。
func (r *knowledgeRepository) List(ctx context.Context, query model.KnowledgeQuery) ([]model.Knowledge, error) {
	var (
		sb   strings.Builder
		args []any
	)

	// 显式列出字段，避免 SELECT * 带来的隐式耦合。
	sb.WriteString("SELECT id, type, title, content, created_at, updated_at FROM knowledge WHERE 1 = 1")
	if query.Type != "" {
		sb.WriteString(" AND type = ?")
		args = append(args, query.Type)
	}
	if query.Title != "" {
		sb.WriteString(" AND title LIKE ?")
		args = append(args, "%"+query.Title+"%")
	}

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query knowledge list: %w", err)
	}
	defer rows.Close() // Close 必须调用，否则数据库连接泄漏。

	items := make([]model.Knowledge, 0)
	for rows.Next() {
		var k model.Knowledge
		if err := rows.Scan(&k.ID, &k.Type, &k.Title, &k.Content, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan knowledge row: %w", err)
		}
		items = append(items, k)
	}
	// 迭代结束后必须检查 rows.Err()，否则可能静默丢数据。
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge rows: %w", err)
	}

	return items, nil
}
