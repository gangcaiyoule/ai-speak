package scene

import (
	"context"
	"errors"
	"reflect"
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

func TestCatalogReadsDoNotExposeMutableState(t *testing.T) {
	for _, method := range []string{"GetScene", "ListScenes", "ListRoles"} {
		t.Run(method, func(t *testing.T) {
			ctx := context.Background()
			catalog := NewCatalog()
			want, err := catalog.GetScene(ctx, "self-introduction")
			if err != nil {
				t.Fatal(err)
			}
			switch method {
			case "GetScene", "ListScenes":
				var value Scene
				if method == "GetScene" {
					value, err = catalog.GetScene(ctx, want.ID)
				} else {
					var values []Scene
					values, err = catalog.ListScenes(ctx)
					if err != nil {
						t.Fatal(err)
					}
					value = values[0]
				}
				if err != nil {
					t.Fatal(err)
				}
				value.Prompt.FocusAreas[0] = "changed"
				value.Prompt.TurnBlueprints[0] = "changed"
				value.Roles[0].DisplayName = "changed"
				value.Roles[0].PracticeObjectives[0].Description = "changed"
				value.PracticeOptions[0].DisplayName = "changed"
				*value.PracticeOptions[1].RoleDefinitionID = "changed"
			case "ListRoles":
				roles, err := catalog.ListRoles(ctx, want.ID)
				if err != nil {
					t.Fatal(err)
				}
				roles[0].DisplayName = "changed"
				roles[0].PracticeObjectives[0].Description = "changed"
			}
			got, err := catalog.GetScene(ctx, want.ID)
			if err != nil {
				t.Fatal(err)
			}
			baseline, err := NewCatalog().GetScene(ctx, want.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, baseline) {
				t.Fatalf("%s mutated catalog: got %#v, want %#v", method, got, baseline)
			}
		})
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
