package graph

import "github.com/masonhuemmer/m365/internal/app/teams"

// mentionPayload builds Graph's mentions array. Each entry pairs with the
// <at id> tag of the same id in the HTML body.
func mentionPayload(ms []teams.Mention) []map[string]any {
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, map[string]any{
			"id":          m.ID,
			"mentionText": m.Name,
			"mentioned": map[string]any{"user": map[string]any{
				"id": m.UserID, "displayName": m.Name, "userIdentityType": "aadUser",
			}},
		})
	}
	return out
}
