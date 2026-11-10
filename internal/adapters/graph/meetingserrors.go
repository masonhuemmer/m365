package graph

// mapMeetingsError maps a failed meetings request to a CLI error.
func mapMeetingsError(status int, body []byte) error {
	return MapGraphError(status, body)
}
