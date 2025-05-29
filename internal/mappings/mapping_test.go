package mappings

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type User struct {
	ID        int
	FirstName string
	LastName  string
}

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserWithTimestamps struct {
	ID        string
	FirstName string
	LastName  string
	Timestamps
}

type Blog struct {
	ID          string
	Title       string
	Description string
	User        User
}

type BlogWithTags struct {
	ID          string `db:"blog_id,pk,generated,autoincr"`
	Title       string `db:"title,pk"`
	Description string `db:"description,generated"`
	User        User   `db:"-"`
}

func TestGetMappings(t *testing.T) {
	testGetMappings[User](t, Mapping{
		All:           []string{"id", "first_name", "last_name"},
		PKs:           make([]string, 3),
		NonPKs:        []string{"id", "first_name", "last_name"},
		Generated:     make([]string, 3),
		NonGenerated:  []string{"id", "first_name", "last_name"},
		AutoIncrement: make([]string, 3),
	})

	testGetMappings[Timestamps](t, Mapping{
		All:           []string{"created_at", "updated_at"},
		PKs:           make([]string, 2),
		NonPKs:        []string{"created_at", "updated_at"},
		Generated:     make([]string, 2),
		NonGenerated:  []string{"created_at", "updated_at"},
		AutoIncrement: make([]string, 2),
	})

	testGetMappings[UserWithTimestamps](t, Mapping{
		All:           []string{"id", "first_name", "last_name", "timestamps"},
		PKs:           make([]string, 4),
		NonPKs:        []string{"id", "first_name", "last_name", "timestamps"},
		Generated:     make([]string, 4),
		NonGenerated:  []string{"id", "first_name", "last_name", "timestamps"},
		AutoIncrement: make([]string, 4),
	})

	testGetMappings[Blog](t, Mapping{
		All:           []string{"id", "title", "description", "user"},
		PKs:           make([]string, 4),
		NonPKs:        []string{"id", "title", "description", "user"},
		Generated:     make([]string, 4),
		NonGenerated:  []string{"id", "title", "description", "user"},
		AutoIncrement: make([]string, 4),
	})

	testGetMappings[BlogWithTags](t, Mapping{
		All:           []string{"blog_id", "title", "description", ""},
		PKs:           []string{"blog_id", "title", "", ""},
		NonPKs:        []string{"", "", "description", ""},
		Generated:     []string{"blog_id", "", "description", ""},
		NonGenerated:  []string{"", "title", "", ""},
		AutoIncrement: []string{"blog_id", "", "", ""},
	})
}

func testGetMappings[T any](t *testing.T, expected Mapping) {
	t.Helper()
	var x T
	xTyp := reflect.TypeOf(x)
	t.Run(xTyp.Name(), func(t *testing.T) {
		cols := GetMappings(xTyp)
		if diff := cmp.Diff(expected, cols); diff != "" {
			t.Fatal(diff)
		}
	})
}
