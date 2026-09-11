package repository

import (
	"context"
	"database/sql"
	"knowledge/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) AddUser(ctx context.Context, user model.User) (bool, error) {
	_, err := r.db.ExecContext(ctx, "INSERT INTO user (username, password, email) VALUES (?, ?, ?)", user.Username, user.Password, user.Email)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepository) QueryUser(ctx context.Context, query model.UserQuery) ([]*model.User, error) {
	sqlStr, args := r.buildSqlParams(query)
	rows, err := r.db.QueryContext(ctx, "select * from user where "+sqlStr, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close() // Close the database connection when the function returns

	var users []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Email)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *UserRepository) buildSqlParams(query model.UserQuery) (sqlStr string, args []any) {
	if query.ID != 0 {
		sqlStr += "id = ?"
		args = append(args, query.ID)
	}
	if query.Username != "" {
		sqlStr += "username = ?"
		args = append(args, query.Username)
	}
	if query.Email != "" {
		sqlStr += "email = ?"
		args = append(args, query.Email)
	}
	return
}

func (r *UserRepository) Login(ctx context.Context, user model.User) (bool, error) {
	row, err := r.db.Query("select *from user where username = ? and password = ?", user.Username, user.Password)
	if err != nil {
		return false, err
	}
	defer row.Close()
	return row.Next(), nil
}
