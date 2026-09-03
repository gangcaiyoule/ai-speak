package scene

import (
	"context"
	"errors"
	"testing"
)

func TestCatalogListScenesPreservesConfiguredOrder(t *testing.T) {
	scenes, err := NewCatalog().ListScenes(context.Background())
	if err != nil {
		t.Fatalf("ListScenes() error = %v", err)
	}
	if len(scenes) != 2 || scenes[0].ID != "self-introduction" || scenes[1].ID != "project-deep-dive" {
		t.Fatalf("unexpected scene order: %#v", scenes)
	}
}

func TestCatalogGetSceneReturnsNotFound(t *testing.T) {
	_, err := NewCatalog().GetScene(context.Background(), "missing")
	if !errors.Is(err, ErrSceneNotFound) {
		t.Fatalf("GetScene() error = %v, want ErrSceneNotFound", err)
	}
}

func TestCatalogListScenesRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewCatalog().ListScenes(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ListScenes() error = %v, want context.Canceled", err)
	}
}
