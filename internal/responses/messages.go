package responses

func GetSinglePieceMessage() string {
	return `
		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" %v />
		</span>
	`
}

func GetEatPiecesMessage() string {
	return `
		<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="display: none">
			<img src="/assets/pieces/%v.svg" />
		</span>

		<span id="%v" hx-post="/move" hx-swap-oob="true" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>

		<div id="lost-pieces-%v" hx-swap-oob="afterbegin">
			<img src="/assets/pieces/%v.svg" class="w-[18px] h-[18px]" />
		</div>
	`
}

func GetReselectPieceMessage() string {
	return `
		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" class="bg-sky-300" />
		</span>
	
		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" %v  />
		</span>
	`
}

func GetCoverCheckMessage() string {
	return `
		<div id="%v" hx-post="/move-to" hx-swap-oob="true" class="tile tile-md h-full w-full" style="background-color: %v"></div>

		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>

		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>
	`
}

func GetTileMessage() string {
	return `
		<div id="%v" hx-post="/%v" hx-swap-oob="true" class="tile tile-md" style="background-color: %v"></div>
	`
}

func GetTimerMessage() string {
	return `
		<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-white">%v</div>
	
		<div id="%v" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>
	`
}

func GetPromotionDoneMessage() string {
	return `
		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>

		<div id="overlay" hx-swap-oob="true" class="hidden w-board w-board-md h-board h-board-md absolute z-20 hover:cursor-default"></div>

		<div id="promotion" hx-swap-oob="true" class="absolute"></div>
	`
}

func GetCastleMessage() string {
	return `
		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>

		<span id="%v" hx-post="/move" hx-swap-oob="true" hx-swap="outerHTML" class="tile tile-md hover:cursor-grab absolute transition-all" style="bottom: %vpx; left: %vpx">
			<img src="/assets/pieces/%v.svg" />
		</span>
	`
}

func GetPromotionInitMessage() string {
	return `
		<div 
			id="promotion"
			hx-swap-oob="true"
			style="%v; left: %vpx"
			class="absolute mt-2 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50 opacity-0 fade-in-opacity"
		>
			<div class="grid gap-2 py-2">
				<img src="/assets/pieces/%v_queen.svg" hx-post="/promotion?pawn=%v&piece=%v_queen" alt="Queen" class="tile tile-md cursor-pointer hover:scale-105 transition opacity-0 fade-in-opacity" />
				<img src="/assets/pieces/%v_rook.svg" hx-post="/promotion?pawn=%v&piece=right_%v_rook" alt="Rook" class="tile tile-md cursor-pointer hover:scale-105 transition opacity-0 fade-in-opacity" />
				<img src="/assets/pieces/%v_knight.svg" hx-post="/promotion?pawn=%v&piece=right_%v_knight" alt="Knight" class="tile tile-md cursor-pointer hover:scale-105 transition opacity-0 fade-in-opacity" />
				<img src="/assets/pieces/%v_bishop.svg" hx-post="/promotion?pawn=%v&piece=right_%v_bishop" alt="Bishop" class="tile tile-md cursor-pointer hover:scale-105 transition opacity-0 fade-in-opacity" />
			</div>
		</div>

		<div id="overlay" hx-swap-oob="true" class="w-board w-board-md h-board h-board-md absolute z-20 hover:cursor-default"></div>
	`
}

func GetMovesUpdateMessage() string {
	return `
		<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 w-[240px] text-white h-moves mt-8">
			<span>%v</span>
		</div>
	`
}

func GetMovesNumberUpdateMessage() string {
	return `
		<div id="moves" hx-swap-oob="beforeend" class="grid grid-cols-3 w-[240px] text-white h-moves mt-8">
			<span>%v.</span>
			<span>%v</span>
		</div>
	`
}

func GetTimePicker() string {
	return `
		<div class="absolute right-0 mt-2 w-48 bg-[#1e1c1a] border border-[#3a3733] text-white rounded-md shadow-lg z-50">
			<div hx-post="/set-time" hx-vals='{"time": "15"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">15 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "15", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">15 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "10"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">10 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "10", "addition": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">10 + 3</div>
			<div hx-post="/set-time" hx-vals='{"time": "3"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">3 Minutes</div>
			<div hx-post="/set-time" hx-vals='{"time": "3", "addition": "1"}' hx-target="#timer" class="block px-4 py-2 hover:bg-emerald-600 hover:text-white transition cursor-pointer">3 + 1</div>
		</div>
	`
}

func GetTimerSwitchMessage() string {
	return `
		<div id="dropdown-menu" hx-swap-oob="true" class="relative mb-8"></div>

		<div id="white" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<div id="black" hx-swap-oob="true" class="px-7 py-3 bg-gray-500">%v</div>

		<input type="hidden" id="timer-value" name="duration" hx-swap-oob="true" value="%v" />

		%v Min %v
	`
}

func GetLogErrorMessage() string {
	return `
		<div id="incorrect-password" hx-swap-oob="innerHTML">
			<p class="text-red-400 text-center">%v</p>
		</div>
	`
}
