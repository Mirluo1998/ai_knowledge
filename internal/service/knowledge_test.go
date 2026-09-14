package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// fakeRepository 是 KnowledgeRepository 的内存实现，
// 验证了“接口定义在消费方”带来的可测试性：无需真实数据库。
type fakeRepository struct {
	items []model.Knowledge
	err   error

	gotCtx   context.Context
	gotQuery model.KnowledgeQuery
}

func (f *fakeRepository) List(ctx context.Context, query model.KnowledgeQuery) ([]model.Knowledge, error) {
	f.gotCtx = ctx
	f.gotQuery = query
	return f.items, f.err
}

func (f *fakeRepository) Create(ctx context.Context, knowledge model.Knowledge) (int32, error) {
	return 1, f.err
}

func TestListKnowledge(t *testing.T) {
	repo := &fakeRepository{items: []model.Knowledge{
		{ID: 1, Type: 1, Title: "部署流程"},
		{ID: 2, Type: 1, Title: "发布规范"},
	}}
	svc := service.NewKnowledgeService(repo)

	ctx := context.WithValue(context.Background(), "trace-id", "abc") //nolint:staticcheck // 仅用于验证 context 透传
	items, err := svc.ListKnowledge(ctx, model.KnowledgeQuery{Type: 1})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if repo.gotQuery.Type != 1 {
		t.Errorf("query not passed through, got %+v", repo.gotQuery)
	}
	if repo.gotCtx != ctx {
		t.Error("context not passed through to repository")
	}
}

func TestListKnowledgeValidationError(t *testing.T) {
	svc := service.NewKnowledgeService(&fakeRepository{})

	_, err := svc.ListKnowledge(context.Background(), model.KnowledgeQuery{
		Title: strings.Repeat("x", 101),
	})

	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *service.ValidationError, got %T (%v)", err, err)
	}
}

func TestListKnowledgeRepositoryError(t *testing.T) {
	dbErr := errors.New("connection refused")
	svc := service.NewKnowledgeService(&fakeRepository{err: dbErr})

	_, err := svc.ListKnowledge(context.Background(), model.KnowledgeQuery{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 底层错误必须被包装保留，便于上层用 errors.Is 判断。
	if !errors.Is(err, dbErr) {
		t.Errorf("repository error not wrapped, got: %v", err)
	}
}
