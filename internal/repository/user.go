package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"knowledge/internal/model"
	"knowledge/internal/service"
	"log/slog"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// maxUserQueryRows 限制用户检索返回的最大行数，防止无条件查询拖出全表。
const maxUserQueryRows = 100

type UserRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewUserRepository(db *sql.DB, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepository) AddUser(ctx context.Context, user model.User) (bool, error) {
	_, err := r.db.ExecContext(ctx, "INSERT INTO user (username, password, email) VALUES (?, ?, ?)", user.Username, user.Password, user.Email)
	if err != nil {
		// 1062 是 MySQL 唯一索引冲突：username/email 已被注册。
		// 转成 service 层定义的哨兵错误，由 service 包装为面向客户端的校验错误。
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return false, fmt.Errorf("add user: %w", service.ErrUserAlreadyExists)
		}
		return false, fmt.Errorf("add user: %w", err)
	}
	return true, nil
}

// QueryUser 根据非空查询字段动态检索用户。
// 只检索可对外暴露的列（不含密码哈希），并强制行数上限。
func (r *UserRepository) QueryUser(ctx context.Context, query model.UserQuery) ([]*model.User, error) {
	var (
		sb   strings.Builder
		args []any
	)

	// 显式列出字段，避免 SELECT * 带来的隐式耦合；WHERE 1 = 1 便于动态拼接条件。
	sb.WriteString("SELECT id, username, email FROM user WHERE 1 = 1")
	if query.ID != 0 {
		sb.WriteString(" AND id = ?")
		args = append(args, query.ID)
	}
	if query.Username != "" {
		sb.WriteString(" AND username = ?")
		args = append(args, query.Username)
	}
	if query.Email != "" {
		sb.WriteString(" AND email = ?")
		args = append(args, query.Email)
	}
	sb.WriteString(" LIMIT ?")
	args = append(args, maxUserQueryRows)

	sqlStr := sb.String()
	r.logger.DebugContext(ctx, "querying users", "sql", sqlStr, "args", args)

	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close() // Close 必须调用，否则数据库连接泄漏。

	users := make([]*model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, &user)
	}
	// 迭代结束后必须检查 rows.Err()，否则可能静默丢数据。
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, nil
}

// GetByUsername 按用户名查询用户（含密码哈希），供登录时做密码校验。
// 用户不存在时返回包装了 sql.ErrNoRows 的错误，调用方可用 errors.Is 判断。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(ctx,
		"SELECT id, username, password, email FROM user WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Email)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}
