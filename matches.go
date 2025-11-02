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
