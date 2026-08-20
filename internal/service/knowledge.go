// Package service 实现知识服务的业务逻辑层。
package service

import (
	"context"
	"fmt"

	"knowledge/internal/model"
)

// KnowledgeRepository 定义 service 层所需的数据访问能力。
// 接口定义在消费方，使 service 可以脱离具体数据库实现进行单元测试。
type KnowledgeRepository interface {
	List(ctx context.Context, query model.KnowledgeQuery) ([]model.Knowledge, error)
	Create(ctx context.Context, knowledge model.Knowledge) (int32, error)
}

// KnowledgeService 负责知识条目相关的业务逻辑。
type KnowledgeService struct {
	repo KnowledgeRepository
}

// NewKnowledgeService 创建知识服务实例。
func NewKnowledgeService(repo KnowledgeRepository) *KnowledgeService {
	return &KnowledgeService{repo: repo}
}

// ListKnowledge 按条件查询知识条目列表。
func (s *KnowledgeService) ListKnowledge(ctx context.Context, query model.KnowledgeQuery) ([]model.Knowledge, error) {
	if len(query.Title) > 100 {
		return nil, &ValidationError{Message: "title filter must not exceed 100 characters"}
	}

	items, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list knowledge: %w", err)
	}
	return items, nil
}

func (s *KnowledgeService) CreateKnowledge(ctx context.Context, knowledge model.Knowledge) (int32, error) {
	rw, err := s.repo.Create(ctx, knowledge)
	if err != nil {
		return 0, fmt.Errorf("create knowledge: %w", err)
	}
	return rw, nil
}
