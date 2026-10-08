package spaces

import "github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"

func RewriteSpace(raw, spaceID string) (string, error) {
	return topics.RewriteSpaceIDs(raw, spaceID)
}
