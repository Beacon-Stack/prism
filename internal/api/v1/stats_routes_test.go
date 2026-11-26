package v1

import (
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

// The watch-sync routes own /api/v1/stats/watch, so the Stats page's
// fuller watching summary must live at a path of its own: registering both
// sets on one API has to leave every path with a single GET.
func TestStatsAndWatchSyncRoutesDontCollide(t *testing.T) {
	_, api := humatest.New(t)
	RegisterStatsRoutes(api, nil)
	RegisterWatchSyncRoutes(api, nil)

	seen := map[string]string{}
	for path, item := range api.OpenAPI().Paths {
		if item.Get == nil {
			continue
		}
		if other, dup := seen[item.Get.OperationID]; dup {
			t.Errorf("operation %s registered at both %s and %s", item.Get.OperationID, other, path)
		}
		seen[item.Get.OperationID] = path
	}
	if seen["get-watch-stats"] != "/api/v1/stats/watch" {
		t.Errorf("watch-sync stats at %q", seen["get-watch-stats"])
	}
	if seen["get-stats-watching"] != "/api/v1/stats/watching" {
		t.Errorf("watching summary at %q", seen["get-stats-watching"])
	}
}
