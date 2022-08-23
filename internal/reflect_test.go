package internal

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type User struct {
	ID        string
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
	ID          string `db:"blog_id,pk,generated"`
	Title       string
	Description string
	User        User `db:"-"`
}

func TestGetColumns(t *testing.T) {
	testGetColumns[User](t, Columns{All: []string{
		"id",
		"first_name",
		"last_name",
	}})

	testGetColumns[Timestamps](t, Columns{All: []string{
		"created_at",
		"updated_at",
	}})

	testGetColumns[UserWithTimestamps](t, Columns{All: []string{
		"id",
		"first_name",
		"last_name",
		"created_at",
		"updated_at",
	}})

	testGetColumns[Blog](t, Columns{All: []string{
		"id",
		"title",
		"description",
		"user",
	}})

	testGetColumns[BlogWithTags](t, Columns{All: []string{
		"blog_id",
		"title",
		"description",
	}})
}

func testGetColumns[T any](t *testing.T, expected Columns) {
	t.Helper()
	var x T
	xTyp := reflect.TypeOf(x)
	t.Run(xTyp.Name(), func(t *testing.T) {
		cols := GetColumns(xTyp)
		if diff := cmp.Diff(expected, cols); diff != "" {
			t.Fatal(diff)
		}
	})
}
