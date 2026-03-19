package storage_minio

import (
	"testing"

	"github.com/infrago/storage"
)

func TestResolveBucket(t *testing.T) {
	cases := []struct {
		name    string
		inst    *storage.Instance
		project string
		want    string
	}{
		{
			name:    "project fallback",
			inst:    &storage.Instance{},
			project: "demo",
			want:    "demo",
		},
		{
			name:    "explicit bucket wins",
			inst:    &storage.Instance{Setting: map[string]interface{}{"bucket": "assets"}},
			project: "demo",
			want:    "assets",
		},
		{
			name:    "empty project fallback to default",
			inst:    &storage.Instance{},
			project: "",
			want:    "default",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveBucket(c.inst, c.project); got != c.want {
				t.Fatalf("want %q got %q", c.want, got)
			}
		})
	}
}
