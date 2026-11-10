package domain

// Transcript is Teams' record of one transcription run in a meeting.
type Transcript struct {
	ID      string `json:"id"`
	Created string `json:"created"`
	Ended   string `json:"ended"`
}

// TranscriptList is the output of meetings transcripts: newest first.
type TranscriptList struct {
	Count int          `json:"count"`
	Items []Transcript `json:"items"`
}

// SavedTranscript summarises a saved transcript. It never holds transcript text.
type SavedTranscript struct {
	Path     string   `json:"path"`
	ID       string   `json:"id"`
	Created  string   `json:"created"`
	Ended    string   `json:"ended"`
	Bytes    int      `json:"bytes"`
	Speakers []string `json:"speakers"`
}
