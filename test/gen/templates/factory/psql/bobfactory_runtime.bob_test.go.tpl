{{- $hasUsers := has "users" $.TableNames -}}
{{- $hasVideos := has "videos" $.TableNames -}}
{{- $hasTags := has "tags" $.TableNames -}}
{{- $hasVideoTags := has "video_tags" $.TableNames -}}
{{- $hasSponsors := has "sponsors" $.TableNames -}}

{{- /* Only generate tests if we have the required tables */}}
{{- if and $hasUsers $hasVideos -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

// =============================================================================
// CRUD Operations Tests
// =============================================================================

// TestUserReload tests that Reload correctly refreshes a model from the database
func TestUserReload(t *testing.T) {
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
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
	originalID := user.ID

	// Reload and verify the ID is still correct
	if err := user.Reload(ctx, tx); err != nil {
		t.Fatalf("Error reloading User: %v", err)
	}
	if user.ID != originalID {
		t.Fatalf("Expected ID %d after reload, got %d", originalID, user.ID)
	}
}

// TestVideoUpdateAndReload tests Update and Reload operations
func TestVideoUpdateAndReload(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user and video
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	video := New().NewVideoWithContext(ctx,
		VideoMods.WithExistingUser(user),
	).CreateOrFail(ctx, t, tx)

	// Create another user to update the video's user_id
	user2 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Update the video to point to user2
	{{$.Importer.Import "github.com/aarondl/opt/omit"}}
	err = video.Update(ctx, tx, &models.VideoSetter{
		UserID: omit.From(user2.ID),
	})
	if err != nil {
		t.Fatalf("Error updating Video: %v", err)
	}

	// Verify the update took effect
	if video.UserID != user2.ID {
		t.Fatalf("Expected UserID %d after update, got %d", user2.ID, video.UserID)
	}

	// Reload and verify
	if err := video.Reload(ctx, tx); err != nil {
		t.Fatalf("Error reloading Video: %v", err)
	}
	if video.UserID != user2.ID {
		t.Fatalf("Expected UserID %d after reload, got %d", user2.ID, video.UserID)
	}
}

// TestVideoDelete tests the Delete operation
func TestVideoDelete(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a video
	video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)
	videoID := video.ID

	// Delete the video
	if err := video.Delete(ctx, tx); err != nil {
		t.Fatalf("Error deleting Video: %v", err)
	}

	// Try to find it - should not exist
	_, err = models.Videos.Query(models.SelectWhere.Videos.ID.EQ(videoID)).One(ctx, tx)
	if err == nil {
		t.Fatal("Expected error when querying deleted video, got nil")
	}
}

// =============================================================================
// Slice Operations Tests
// =============================================================================

// TestUserSliceReloadAll tests ReloadAll on a slice
func TestUserSliceReloadAll(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create multiple users
	users := New().NewUserWithContext(ctx).CreateManyOrFail(ctx, t, tx, 3)

	// Store original IDs
	originalIDs := make([]int32, len(users))
	for i, u := range users {
		originalIDs[i] = u.ID
	}

	// Reload all
	if err := users.ReloadAll(ctx, tx); err != nil {
		t.Fatalf("Error reloading users: %v", err)
	}

	// Verify IDs are still correct
	for i, u := range users {
		if u.ID != originalIDs[i] {
			t.Fatalf("Expected ID %d after reload, got %d", originalIDs[i], u.ID)
		}
	}
}

// TestVideoSliceDeleteAll tests DeleteAll on a slice
func TestVideoSliceDeleteAll(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user and multiple videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	var videos models.VideoSlice
	for i := 0; i < 3; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
		videos = append(videos, video)
	}

	// Delete all videos
	if err := videos.DeleteAll(ctx, tx); err != nil {
		t.Fatalf("Error deleting videos: %v", err)
	}

	// Verify user has no videos
	if err := user.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *user.C.Videos != 0 {
		t.Fatalf("Expected 0 videos after delete, got %d", *user.C.Videos)
	}
}

// =============================================================================
// Relationship Loading Tests
// =============================================================================

// TestLoadUserFromVideo tests loading the User relationship from a Video
func TestLoadUserFromVideo(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user and video
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	video := New().NewVideoWithContext(ctx,
		VideoMods.WithExistingUser(user),
	).CreateOrFail(ctx, t, tx)

	// Load the user relationship
	if err := video.LoadUser(ctx, tx); err != nil {
		t.Fatalf("Error loading User: %v", err)
	}

	// Verify the loaded user
	if video.R.User == nil {
		t.Fatal("Expected User to be loaded, got nil")
	}
	if video.R.User.ID != user.ID {
		t.Fatalf("Expected User ID %d, got %d", user.ID, video.R.User.ID)
	}
}

// TestLoadVideosFromUser tests loading the Videos relationship from a User
func TestLoadVideosFromUser(t *testing.T) {
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
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create videos for the user
	for i := 0; i < 3; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
	}

	// Load the videos relationship
	if err := user.LoadVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading Videos: %v", err)
	}

	// Verify the loaded videos
	if len(user.R.Videos) != 3 {
		t.Fatalf("Expected 3 videos, got %d", len(user.R.Videos))
	}
}

// TestLoadVideosFromUserSlice tests loading Videos for multiple Users at once
func TestLoadVideosFromUserSlice(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users with different numbers of videos
	var users models.UserSlice
	expectedCounts := []int{1, 2, 3}
	for _, count := range expectedCounts {
		user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
		for j := 0; j < count; j++ {
			New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).CreateOrFail(ctx, t, tx)
		}
		users = append(users, user)
	}

	// Load videos for all users at once
	if err := users.LoadVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading Videos: %v", err)
	}

	// Verify each user has the correct number of videos
	for i, user := range users {
		if len(user.R.Videos) != expectedCounts[i] {
			t.Fatalf("Expected user %d to have %d videos, got %d", i, expectedCounts[i], len(user.R.Videos))
		}
	}
}

// =============================================================================
// Relationship Attachment Tests
// =============================================================================

// TestInsertVideosForUser tests InsertVideos to create and attach videos
func TestInsertVideosForUser(t *testing.T) {
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
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Insert videos using InsertVideos
	err = user.InsertVideos(ctx, tx, &models.VideoSetter{}, &models.VideoSetter{})
	if err != nil {
		t.Fatalf("Error inserting Videos: %v", err)
	}

	// Verify the videos were created
	if err := user.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *user.C.Videos != 2 {
		t.Fatalf("Expected 2 videos, got %d", *user.C.Videos)
	}
}

// TestAttachVideosToUser tests AttachVideos to attach existing videos
func TestAttachVideosToUser(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create two users
	user1 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
	user2 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create videos for user1
	var videos []*models.Video
	for i := 0; i < 2; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user1),
		).CreateOrFail(ctx, t, tx)
		videos = append(videos, video)
	}

	// Attach videos to user2
	if err := user2.AttachVideos(ctx, tx, videos...); err != nil {
		t.Fatalf("Error attaching Videos: %v", err)
	}

	// Verify user2 now has the videos
	if err := user2.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *user2.C.Videos != 2 {
		t.Fatalf("Expected 2 videos for user2, got %d", *user2.C.Videos)
	}

	// Verify user1 no longer has the videos
	if err := user1.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *user1.C.Videos != 0 {
		t.Fatalf("Expected 0 videos for user1, got %d", *user1.C.Videos)
	}
}

// =============================================================================
// Count Loading Tests
// =============================================================================

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
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

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
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
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
	users := New().NewUserWithContext(ctx).CreateManyOrFail(ctx, t, tx, 2)

	// Create different number of videos for each user
	for i, user := range users {
		numVideos := i + 1 // First user gets 1, second gets 2
		for j := 0; j < numVideos; j++ {
			New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).CreateOrFail(ctx, t, tx)
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

// TestPreloadCountUserVideos tests using PreloadCount to load counts in the same query
func TestPreloadCountUserVideos(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users with different numbers of videos
	expectedCounts := []int{2, 3, 1}
	for _, count := range expectedCounts {
		user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
		for j := 0; j < count; j++ {
			New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).CreateOrFail(ctx, t, tx)
		}
	}

	// Query users with PreloadCount to load video counts in the same query
	users, err := models.Users.Query(
		models.PreloadCount.User.Videos(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with PreloadCount: %v", err)
	}

	if len(users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(users))
	}

	// Verify counts are loaded
	for i, user := range users {
		if user.C.Videos == nil {
			t.Fatalf("Expected Videos count to be set for user %d, got nil", i)
		}
		expectedCount := int64(expectedCounts[i])
		if *user.C.Videos != expectedCount {
			t.Fatalf("Expected Videos count for user %d to be %d, got %d", i, expectedCount, *user.C.Videos)
		}
	}
}

// TestThenLoadCountUserVideos tests using ThenLoadCount to load counts in a separate query
func TestThenLoadCountUserVideos(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users with different numbers of videos
	expectedCounts := []int{2, 3, 1}
	for _, count := range expectedCounts {
		user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
		for j := 0; j < count; j++ {
			New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).CreateOrFail(ctx, t, tx)
		}
	}

	// Query users with ThenLoadCount to load video counts in a separate query
	users, err := models.Users.Query(
		models.ThenLoadCount.User.Videos(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with ThenLoadCount: %v", err)
	}

	if len(users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(users))
	}

	// Verify counts are loaded
	for i, user := range users {
		if user.C.Videos == nil {
			t.Fatalf("Expected Videos count to be set for user %d, got nil", i)
		}
		expectedCount := int64(expectedCounts[i])
		if *user.C.Videos != expectedCount {
			t.Fatalf("Expected Videos count for user %d to be %d, got %d", i, expectedCount, *user.C.Videos)
		}
	}
}

// TestPreloadCountWithFilter tests PreloadCount with filtering mods
func TestPreloadCountWithFilter(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user with multiple videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
	
	// Create 5 videos with different IDs
	var videoIDs []int32
	for i := 0; i < 5; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
		videoIDs = append(videoIDs, video.ID)
	}

	// Query user with PreloadCount filtering for videos with ID > first video ID
	// This should count only 4 videos
	users, err := models.Users.Query(
		models.SelectWhere.Users.ID.EQ(user.ID),
		models.PreloadCount.User.Videos(
			models.SelectWhere.Videos.ID.GT(videoIDs[0]),
		),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with filtered PreloadCount: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	// Verify filtered count
	if users[0].C.Videos == nil {
		t.Fatal("Expected Videos count to be set, got nil")
	}
	if *users[0].C.Videos != 4 {
		t.Fatalf("Expected filtered Videos count to be 4, got %d", *users[0].C.Videos)
	}
}

// TestThenLoadCountWithFilter tests ThenLoadCount with filtering mods
func TestThenLoadCountWithFilter(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user with multiple videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
	
	// Create 5 videos with different IDs
	var videoIDs []int32
	for i := 0; i < 5; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
		videoIDs = append(videoIDs, video.ID)
	}

	// Query user with ThenLoadCount filtering for videos with ID > first video ID
	// This should count only 4 videos
	users, err := models.Users.Query(
		models.SelectWhere.Users.ID.EQ(user.ID),
		models.ThenLoadCount.User.Videos(
			models.SelectWhere.Videos.ID.GT(videoIDs[0]),
		),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with filtered ThenLoadCount: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	// Verify filtered count
	if users[0].C.Videos == nil {
		t.Fatal("Expected Videos count to be set, got nil")
	}
	if *users[0].C.Videos != 4 {
		t.Fatalf("Expected filtered Videos count to be 4, got %d", *users[0].C.Videos)
	}
}
{{- end }}

{{- if and $hasVideos $hasTags $hasVideoTags }}
// =============================================================================
// Many-to-Many Relationship Tests
// =============================================================================

// TestManyToManyVideoTags tests the many-to-many relationship between Videos and Tags
func TestManyToManyVideoTags(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a video
	video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create tags and attach them
	tags := New().NewTagWithContext(ctx).CreateManyOrFail(ctx, t, tx, 3)

	// Attach all tags to video
	if err := video.AttachTags(ctx, tx, tags...); err != nil {
		t.Fatalf("Error attaching Tags: %v", err)
	}

	// Load tags and verify
	if err := video.LoadTags(ctx, tx); err != nil {
		t.Fatalf("Error loading Tags: %v", err)
	}
	if len(video.R.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(video.R.Tags))
	}

	// Verify count
	if err := video.LoadCountTags(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *video.C.Tags != 3 {
		t.Fatalf("Expected count 3, got %d", *video.C.Tags)
	}
}

// TestSharedTagsAcrossVideos tests that tags can be shared across multiple videos
func TestSharedTagsAcrossVideos(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a shared tag
	sharedTag := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create multiple videos and attach the shared tag to each
	for i := 0; i < 3; i++ {
		video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)
		if err := video.AttachTags(ctx, tx, sharedTag); err != nil {
			t.Fatalf("Error attaching Tag: %v", err)
		}
	}

	// Verify the tag is associated with 3 videos
	if err := sharedTag.LoadCountVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading count: %v", err)
	}
	if *sharedTag.C.Videos != 3 {
		t.Fatalf("Expected tag to have 3 videos, got %d", *sharedTag.C.Videos)
	}

	// Load and verify
	if err := sharedTag.LoadVideos(ctx, tx); err != nil {
		t.Fatalf("Error loading Videos: %v", err)
	}
	if len(sharedTag.R.Videos) != 3 {
		t.Fatalf("Expected 3 videos, got %d", len(sharedTag.R.Videos))
	}
}

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

	// Create a video
	video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)

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
		tag := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)
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
	tag := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)

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
		video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)
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

{{- if and $hasVideos $hasSponsors }}
// =============================================================================
// Optional Relationship Tests (nullable foreign key)
// =============================================================================

// TestOptionalSponsorRelationship tests the optional sponsor relationship on Video
func TestOptionalSponsorRelationship(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a video without a sponsor
	video := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Verify video has no sponsor_id set
	if _, ok := video.SponsorID.Get(); ok {
		t.Fatal("Expected SponsorID to be unset for video without sponsor")
	}

	// Create a sponsor and attach it to the video
	sponsor := New().NewSponsorWithContext(ctx).CreateOrFail(ctx, t, tx)

	if err := video.AttachSponsor(ctx, tx, sponsor); err != nil {
		t.Fatalf("Error attaching Sponsor: %v", err)
	}

	// Reload and verify sponsor_id is now set
	if err := video.Reload(ctx, tx); err != nil {
		t.Fatalf("Error reloading Video: %v", err)
	}
	if _, ok := video.SponsorID.Get(); !ok {
		t.Fatal("Expected SponsorID to be set after attach")
	}

	// Load sponsor and verify it's correct
	if err := video.LoadSponsor(ctx, tx); err != nil {
		t.Fatalf("Error loading Sponsor: %v", err)
	}
	if video.R.Sponsor == nil {
		t.Fatal("Expected Sponsor to be loaded")
	}
	if video.R.Sponsor.ID != sponsor.ID {
		t.Fatalf("Expected Sponsor ID %d, got %d", sponsor.ID, video.R.Sponsor.ID)
	}
}
{{- end }}

{{- if and $hasUsers $hasVideos }}
// =============================================================================
// Query Building with Mods Tests
// =============================================================================

// TestQueryWithWhere tests using Where mods to filter queries
func TestQueryWithWhere(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a user and multiple videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	var videoIDs []int32
	for i := 0; i < 5; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
		videoIDs = append(videoIDs, video.ID)
	}

	// Query videos using Where clause
	videos, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user.ID),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos: %v", err)
	}

	if len(videos) != 5 {
		t.Fatalf("Expected 5 videos, got %d", len(videos))
	}

	// Query with multiple Where conditions
	{{$.Importer.Import "sm" "github.com/stephenafamo/bob/dialect/psql/sm"}}
	videos, err = models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user.ID),
		models.SelectWhere.Videos.ID.GT(videoIDs[0]),
		sm.Limit(3),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos with limit: %v", err)
	}

	if len(videos) != 3 {
		t.Fatalf("Expected 3 videos with limit, got %d", len(videos))
	}
}

// TestQueryWithOrderBy tests using OrderBy mods
func TestQueryWithOrderBy(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users
	createdUsers := New().NewUserWithContext(ctx).CreateManyOrFail(ctx, t, tx, 3)
	var userIDs []int32
	for _, user := range createdUsers {
		userIDs = append(userIDs, user.ID)
	}

	// Query with ORDER BY DESC
	{{$.Importer.Import "sm" "github.com/stephenafamo/bob/dialect/psql/sm"}}
	users, err := models.Users.Query(
		sm.OrderBy(models.Users.Columns.ID).Desc(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users: %v", err)
	}

	// Verify descending order
	for i := 0; i < len(users)-1; i++ {
		if users[i].ID < users[i+1].ID {
			t.Fatal("Expected users in descending order by ID")
		}
	}

	// Query with ORDER BY ASC
	users, err = models.Users.Query(
		sm.OrderBy(models.Users.Columns.ID).Asc(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users: %v", err)
	}

	// Verify ascending order
	for i := 0; i < len(users)-1; i++ {
		if users[i].ID > users[i+1].ID {
			t.Fatal("Expected users in ascending order by ID")
		}
	}
}

// TestQueryWithLimitOffset tests using Limit and Offset mods
func TestQueryWithLimitOffset(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create 10 users
	New().NewUserWithContext(ctx).CreateManyOrFail(ctx, t, tx, 10)

	// Query with limit
	{{$.Importer.Import "sm" "github.com/stephenafamo/bob/dialect/psql/sm"}}
	users, err := models.Users.Query(
		sm.Limit(5),
		sm.OrderBy(models.Users.Columns.ID).Asc(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users: %v", err)
	}

	if len(users) != 5 {
		t.Fatalf("Expected 5 users with limit, got %d", len(users))
	}

	// Query with limit and offset
	users2, err := models.Users.Query(
		sm.Limit(5),
		sm.Offset(5),
		sm.OrderBy(models.Users.Columns.ID).Asc(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with offset: %v", err)
	}

	if len(users2) != 5 {
		t.Fatalf("Expected 5 users with offset, got %d", len(users2))
	}

	// Verify no overlap between the two result sets
	for _, u1 := range users {
		for _, u2 := range users2 {
			if u1.ID == u2.ID {
				t.Fatalf("Found duplicate user ID %d in offset results", u1.ID)
			}
		}
	}
}

// TestQueryWithIN tests using IN clause
func TestQueryWithIN(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users
	var targetIDs []int32
	for i := 0; i < 5; i++ {
		user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
		if i%2 == 0 {
			targetIDs = append(targetIDs, user.ID)
		}
	}

	// Query using IN clause
	users, err := models.Users.Query(
		models.SelectWhere.Users.ID.In(targetIDs...),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users: %v", err)
	}

	if len(users) != len(targetIDs) {
		t.Fatalf("Expected %d users, got %d", len(targetIDs), len(users))
	}

	// Verify all returned users are in targetIDs
	for _, user := range users {
		found := false
		for _, id := range targetIDs {
			if user.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("User ID %d not in target IDs", user.ID)
		}
	}
}

// TestQueryWithJoin tests using Join mods
func TestQueryWithJoin(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create user and videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	for i := 0; i < 3; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
	}

	// Query videos with join to users
	{{$.Importer.Import "sm" "github.com/stephenafamo/bob/dialect/psql/sm"}}
	videos, err := models.Videos.Query(
		sm.InnerJoin(models.Users.Name()).On(
			models.Videos.Columns.UserID.EQ(models.Users.Columns.ID),
		),
		models.SelectWhere.Users.ID.EQ(user.ID),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos with join: %v", err)
	}

	if len(videos) != 3 {
		t.Fatalf("Expected 3 videos, got %d", len(videos))
	}
}

// TestQueryCount tests using Count queries
func TestQueryCount(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create user and videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	expectedCount := 7
	for i := 0; i < expectedCount; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
	}

	// Count videos for user
	count, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user.ID),
	).Count(ctx, tx)
	if err != nil {
		t.Fatalf("Error counting videos: %v", err)
	}

	if count != int64(expectedCount) {
		t.Fatalf("Expected count %d, got %d", expectedCount, count)
	}
}

// TestQueryExists tests using Exists queries
func TestQueryExists(t *testing.T) {
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
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Check if user exists
	exists, err := models.Users.Query(
		models.SelectWhere.Users.ID.EQ(user.ID),
	).Exists(ctx, tx)
	if err != nil {
		t.Fatalf("Error checking user exists: %v", err)
	}

	if !exists {
		t.Fatal("Expected user to exist")
	}

	// Check for non-existent user
	exists, err = models.Users.Query(
		models.SelectWhere.Users.ID.EQ(999999),
	).Exists(ctx, tx)
	if err != nil {
		t.Fatalf("Error checking non-existent user: %v", err)
	}

	if exists {
		t.Fatal("Expected user to not exist")
	}
}

// TestUpdateWithMods tests using Update with mods
func TestUpdateWithMods(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users and videos
	user1 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	user2 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create videos for user1
	for i := 0; i < 3; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user1),
		).CreateOrFail(ctx, t, tx)
	}

	// Update all videos to belong to user2
	{{$.Importer.Import "github.com/aarondl/opt/omit"}}
	videos, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user1.ID),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos: %v", err)
	}

	err = videos.UpdateAll(ctx, tx, models.VideoSetter{
		UserID: omit.From(user2.ID),
	})
	if err != nil {
		t.Fatalf("Error updating videos: %v", err)
	}

	// Verify user1 has no videos
	count, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user1.ID),
	).Count(ctx, tx)
	if err != nil {
		t.Fatalf("Error counting user1 videos: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 videos for user1, got %d", count)
	}

	// Verify user2 has 3 videos
	count, err = models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user2.ID),
	).Count(ctx, tx)
	if err != nil {
		t.Fatalf("Error counting user2 videos: %v", err)
	}
	if count != 3 {
		t.Fatalf("Expected 3 videos for user2, got %d", count)
	}
}

// TestDeleteWithMods tests using Delete with mods
func TestDeleteWithMods(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create user and videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	var videoIDs []int32
	for i := 0; i < 5; i++ {
		video := New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
		videoIDs = append(videoIDs, video.ID)
	}

	// Delete videos with ID > first video ID
	videos, err := models.Videos.Query(
		models.SelectWhere.Videos.ID.GT(videoIDs[0]),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos: %v", err)
	}

	err = videos.DeleteAll(ctx, tx)
	if err != nil {
		t.Fatalf("Error deleting videos: %v", err)
	}

	// Verify only 1 video remains
	count, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user.ID),
	).Count(ctx, tx)
	if err != nil {
		t.Fatalf("Error counting videos: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected 1 video remaining, got %d", count)
	}
}

// TestSelectJoins tests using SelectJoins with InnerJoin to filter results
func TestSelectJoins(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create two users
	user1 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	user2 := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create videos for both users
	for i := 0; i < 3; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user1),
		).CreateOrFail(ctx, t, tx)
	}

	for i := 0; i < 2; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user2),
		).CreateOrFail(ctx, t, tx)
	}

	// Query videos using SelectJoins with InnerJoin on User
	// The where condition for the joined table is passed at the same level
	videos, err := models.Videos.Query(
		models.SelectJoins.Videos.InnerJoin.User,
		models.SelectWhere.Users.ID.EQ(user1.ID),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos with SelectJoins: %v", err)
	}

	// Should only get user1's videos due to the join condition
	if len(videos) != 3 {
		t.Fatalf("Expected 3 videos for user1, got %d", len(videos))
	}

	// Verify all videos belong to user1
	for i, video := range videos {
		if video.UserID != user1.ID {
			t.Fatalf("Expected video %d to belong to user1 (ID %d), got user ID %d", i, user1.ID, video.UserID)
		}
	}
}

// TestPreloadToOne tests using Preload for to-one relationships
func TestPreloadToOne(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create user and videos
	user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)

	for i := 0; i < 3; i++ {
		New().NewVideoWithContext(ctx,
			VideoMods.WithExistingUser(user),
		).CreateOrFail(ctx, t, tx)
	}

	// Query videos with Preload to eager load User (to-one relationship)
	videos, err := models.Videos.Query(
		models.SelectWhere.Videos.UserID.EQ(user.ID),
		models.Preload.Video.User(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos with Preload: %v", err)
	}

	if len(videos) != 3 {
		t.Fatalf("Expected 3 videos, got %d", len(videos))
	}

	// Verify User is preloaded for each video
	for i, video := range videos {
		if video.R.User == nil {
			t.Fatalf("Expected User to be preloaded for video %d, got nil", i)
		}
		if video.R.User.ID != user.ID {
			t.Fatalf("Expected User ID %d for video %d, got %d", user.ID, i, video.R.User.ID)
		}
	}
}

// TestThenLoadToMany tests using ThenLoad for to-many relationships
func TestThenLoadToMany(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create users with videos
	var users models.UserSlice
	expectedCounts := []int{2, 3, 1}
	for _, count := range expectedCounts {
		user := New().NewUserWithContext(ctx).CreateOrFail(ctx, t, tx)
		for j := 0; j < count; j++ {
			New().NewVideoWithContext(ctx,
				VideoMods.WithExistingUser(user),
			).CreateOrFail(ctx, t, tx)
		}
		users = append(users, user)
	}

	// Query users with ThenLoad to eager load Videos (to-many relationship)
	users, err = models.Users.Query(
		models.SelectThenLoad.User.Videos(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying users with ThenLoad: %v", err)
	}

	if len(users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(users))
	}

	// Verify Videos are preloaded for each user
	for i, user := range users {
		if user.R.Videos == nil {
			t.Fatalf("Expected Videos to be preloaded for user %d, got nil", i)
		}
		if len(user.R.Videos) != expectedCounts[i] {
			t.Fatalf("Expected user %d to have %d videos, got %d", i, expectedCounts[i], len(user.R.Videos))
		}
	}
}
{{- end }}

{{- if and $hasVideos $hasTags $hasVideoTags }}
// TestThenLoadManyToMany tests using ThenLoad for many-to-many relationships
func TestThenLoadManyToMany(t *testing.T) {
	if testDB == nil {
		t.Skip("skipping test, no DSN provided")
	}

	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create videos with tags
	video1 := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)

	video2 := New().NewVideoWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Create tags and attach to videos
	tag1 := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)

	tag2 := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)

	tag3 := New().NewTagWithContext(ctx).CreateOrFail(ctx, t, tx)

	// Attach tags to video1
	if err := video1.AttachTags(ctx, tx, tag1, tag2); err != nil {
		t.Fatalf("Error attaching tags to video1: %v", err)
	}

	// Attach tags to video2
	if err := video2.AttachTags(ctx, tx, tag2, tag3); err != nil {
		t.Fatalf("Error attaching tags to video2: %v", err)
	}

	// Query videos with ThenLoad to eager load Tags (many-to-many)
	videos, err := models.Videos.Query(
		models.SelectThenLoad.Video.Tags(),
	).All(ctx, tx)
	if err != nil {
		t.Fatalf("Error querying videos with ThenLoad: %v", err)
	}

	// Find our videos in the results
	var loadedVideo1, loadedVideo2 *models.Video
	for _, v := range videos {
		if v.ID == video1.ID {
			loadedVideo1 = v
		}
		if v.ID == video2.ID {
			loadedVideo2 = v
		}
	}

	if loadedVideo1 == nil || loadedVideo2 == nil {
		t.Fatal("Expected to find both videos in results")
	}

	// Verify Tags are preloaded
	if loadedVideo1.R.Tags == nil {
		t.Fatal("Expected Tags to be preloaded for video1, got nil")
	}
	if len(loadedVideo1.R.Tags) != 2 {
		t.Fatalf("Expected video1 to have 2 tags, got %d", len(loadedVideo1.R.Tags))
	}

	if loadedVideo2.R.Tags == nil {
		t.Fatal("Expected Tags to be preloaded for video2, got nil")
	}
	if len(loadedVideo2.R.Tags) != 2 {
		t.Fatalf("Expected video2 to have 2 tags, got %d", len(loadedVideo2.R.Tags))
	}
}

{{- end }}
