package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anhdnyopaz/go_victorialog/internal/logger"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserService struct {
	logger logger.Logger
}

func NewUserService(logger logger.Logger) *UserService {
	return &UserService{
		logger: logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	start := time.Now()

	s.logger.Info(ctx, "Create new User", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"action":   "create_user_start",
	})
	time.Sleep(100 * time.Millisecond)
	//Simulate
	if user.Username == "invalid" {
		s.logger.Error(ctx, "Failed to create user", map[string]interface{}{
			"user_id":  user.ID,
			"username": user.Username,
			"action":   "create_user_error",
			"duration": time.Since(start).Milliseconds(),
		})

		return fmt.Errorf("failed to create user")
	}
	s.logger.Info(ctx, "Create new User", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "create_user_success",
		"duration": time.Since(start).Milliseconds(),
	})

	return nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	start := time.Now()
	s.logger.Debug(ctx, "Get User", map[string]interface{}{
		"user_id": id,
		"action":  "get_user_start",
	})
	time.Sleep(100 * time.Millisecond)

	user := &User{
		ID:       id,
		Username: "Demo User",
		Email:    "demo@example.com",
	}

	s.logger.Info(ctx, "Get User", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "get_user_success",
		"duration": time.Since(start).Milliseconds(),
	})
	return user, nil
}
