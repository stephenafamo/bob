-- SelectVideos
SELECT videos.* FROM videos
WHERE id IN ($1)
