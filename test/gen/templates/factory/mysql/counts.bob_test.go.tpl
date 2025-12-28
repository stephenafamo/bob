{{- $hasUsers := $.Tables.Get "users" -}}
{{- $hasVideos := $.Tables.Get "videos" -}}
{{- $hasTags := $.Tables.Get "tags" -}}
{{- $hasVideoTags := $.Tables.Get "video_tags" -}}

{{- /* Only generate tests if we have the required tables */}}
{{- if and $hasUsers $hasVideos -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

// TestLoadCountUserVideos tests that LoadCountVideos works correctly for User
func TestLoadCountUserVideos(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()

	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user
	user, err := New().NewUserWithContext(ctx).Create(ctx, tx)
	if err != nil {
		t.Fatalf("Error creating User: %v", err)
	}

	// Verify initial count is 0
	if err := user.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Videos: %v", err)
	}
	if user.C.Videos == nil {
		t.Fatal("Expected Videos count to be set, got nil")
	}
	if *user.C.Videos != 0 {
		t.Fatalf("Expected Videos count to be 0, got %d", *user.C.Videos)
	}

	// Create 3 videos for this user
	for i := 0; i < 3; i++ {
		_, err := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).Create(ctx, tx)
		if err != nil {
			t.Fatalf("Error creating Video: %v", err)
		}
	}

	// Verify count is now 3
	if err := user.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Videos: %v", err)
	}
	if *user.C.Videos != 3 {
		t.Fatalf("Expected Videos count to be 3, got %d", *user.C.Videos)
	}
}

// TestLoadCountUserVideosSlice tests that LoadCountVideos works correctly for UserSlice
func TestLoadCountUserVideosSlice(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()

	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create 2 users
	var users models.UserSlice
	for i := 0; i < 2; i++ {
		user, err := New().NewUserWithContext(ctx).Create(ctx, tx)
		if err != nil {
			t.Fatalf("Error creating User: %v", err)
		}
		users = append(users, user)
	}

	// Create different number of videos for each user
	for i, user := range users {
		numVideos := i + 1 // First user gets 1, second gets 2
		for j := 0; j < numVideos; j++ {
			_, err := New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).Create(ctx, tx)
			if err != nil {
				t.Fatalf("Error creating Video: %v", err)
			}
		}
	}

	// Load counts for all users
	if err := users.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Videos: %v", err)
	}

	// Verify counts
	for i, user := range users {
		expectedCount := int64(i + 1)
		if user.C.Videos == nil {
			t.Fatalf("Expected Videos count to be set for user %d, got nil", i)
		}
		if *user.C.Videos != expectedCount {
			t.Fatalf("Expected Videos count for user %d to be %d, got %d", i, expectedCount, *user.C.Videos)
		}
	}
}
{{- end }}

{{- if and $hasVideos $hasTags $hasVideoTags }}
// TestLoadCountVideoTags tests that LoadCountTags works correctly for Video
func TestLoadCountVideoTags(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()

	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a video (which will auto-create a user)
	video, err := New().NewVideoWithContext(ctx).Create(ctx, tx)
	if err != nil {
		t.Fatalf("Error creating Video: %v", err)
	}

	// Verify initial count is 0
	if err := video.LoadCountTags(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Tags: %v", err)
	}
	if video.C.Tags == nil {
		t.Fatal("Expected Tags count to be set, got nil")
	}
	if *video.C.Tags != 0 {
		t.Fatalf("Expected Tags count to be 0, got %d", *video.C.Tags)
	}

	// Create 3 tags and attach them to the video
	for i := 0; i < 3; i++ {
		tag, err := New().NewTagWithContext(ctx).Create(ctx, tx)
		if err != nil {
			t.Fatalf("Error creating Tag: %v", err)
		}
		if err := video.AttachTags(ctx, tx, tag); err != nil {
			t.Fatalf("Error attaching Tag to Video: %v", err)
		}
	}

	// Verify count is now 3
	if err := video.LoadCountTags(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Tags: %v", err)
	}
	if *video.C.Tags != 3 {
		t.Fatalf("Expected Tags count to be 3, got %d", *video.C.Tags)
	}
}

// TestLoadCountTagVideos tests that LoadCountVideos works correctly for Tag
func TestLoadCountTagVideos(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()

	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a tag
	tag, err := New().NewTagWithContext(ctx).Create(ctx, tx)
	if err != nil {
		t.Fatalf("Error creating Tag: %v", err)
	}

	// Verify initial count is 0
	if err := tag.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Videos: %v", err)
	}
	if tag.C.Videos == nil {
		t.Fatal("Expected Videos count to be set, got nil")
	}
	if *tag.C.Videos != 0 {
		t.Fatalf("Expected Videos count to be 0, got %d", *tag.C.Videos)
	}

	// Create 3 videos and attach them to the tag
	for i := 0; i < 3; i++ {
		video, err := New().NewVideoWithContext(ctx).Create(ctx, tx)
		if err != nil {
			t.Fatalf("Error creating Video: %v", err)
		}
		if err := tag.AttachVideos(ctx, tx, video); err != nil {
			t.Fatalf("Error attaching Video to Tag: %v", err)
		}
	}

	// Verify count is now 3
	if err := tag.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count for Videos: %v", err)
	}
	if *tag.C.Videos != 3 {
		t.Fatalf("Expected Videos count to be 3, got %d", *tag.C.Videos)
	}
}
{{- end }}
