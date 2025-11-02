package main

func (m *Matches) getMatch(key string) (Match, bool) {
	match, ok := m.matches[key]
	return match, ok
}

func (m *Matches) setMatch(key string, match Match) {
	m.matches[key] = match
}

func (m *Matches) getInitialMatch() Match {
	match := m.matches["initial"]
	return match
}

func (m *Matches) getAllOnlineMatches() map[string]Match {
	onlineMatches := make(map[string]Match)

	for name, match := range m.matches {
		if match.isOnline {
			onlineMatches[name] = match
		}
	}

	return onlineMatches
}

func (m *Match) isOnlineMatch() (OnlineGame, bool) {
	return m.online, m.isOnline
}
