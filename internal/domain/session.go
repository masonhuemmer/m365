package domain

type Session struct {
	SignedIn          bool
	SessionUsable     bool
	Account           string
	MailConsented     bool
	TeamsConsented    bool
	CalendarConsented bool
	FilesConsented    bool
	MeetingsConsented bool
}

func SignedOut() Session {
	return Session{}
}
