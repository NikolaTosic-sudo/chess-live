package utils

import "testing"

func TestReplaceStyles(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		bottom     []int
		left       []int
		wantResult string
	}{
		{
			name: "Replace styles for a single component",
			text: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			bottom: []int{400},
			left:   []int{400},
			wantResult: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 400px; left: 400px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
		},
		{
			name: "Replace styles for 2 components",
			text: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>

				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 200px; left: 200px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
			bottom: []int{400, 600},
			left:   []int{400, 600},
			wantResult: `
				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 400px; left: 400px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>

				<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: 600px; left: 600px">
					<img src="/assets/pieces/%v.svg" %v />
				</span>
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacedText := ReplaceStyles(tt.text, tt.bottom, tt.left)

			if replacedText != tt.wantResult {
				t.Errorf("replaceStyles() replacedText = %v, want %v", replacedText, tt.wantResult)
			}
		})
	}
}
