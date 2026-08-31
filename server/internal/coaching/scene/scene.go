// Package scene 定义口语练习场景相关契约。
package scene

import "context"

// Scene 表示一个可选择的口语练习场景。
type Scene struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Repository 定义场景目录的读取能力。
type Repository interface {
	// List 读取全部可用场景。
	List(context.Context) ([]Scene, error)
	// FindByID 按场景标识读取场景。
	FindByID(context.Context, string) (Scene, error)
}

// Service 定义场景查询用例。
type Service interface {
	// ListScenes 读取全部可用场景。
	ListScenes(context.Context) ([]Scene, error)
}
