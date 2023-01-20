export function word_count(s) {
	s = s.replaceAll(/\p{P}|\p{Sc}/gu, "").trim();
	if ( s === "" ) {
		return 0;
	}
	return s.split(/\s+/).length;
}

export const WORD_TARGET = 5;//750;

(async () => {
	for ( let editor of document.querySelectorAll(".journal-entry") ) {
		((editor) => {
			let textarea = editor.querySelector("textarea");
			let indicator = editor.querySelector(".indicator-tray .wordcount");

			if ( !textarea || !indicator ) {
				return;
			}

			textarea.addEventListener("input", (e) => {
				let wc = word_count(textarea.value);
				if ( wc === 0 ) {
					indicator.textContent = ``;
				} else if ( wc === 1 ) {
					indicator.textContent = `1 word`;
				} else {
					indicator.textContent = `${wc} words`;
				}

				textarea.classList.toggle("-success", wc >= WORD_TARGET);
			});
		})(editor);
	}
})();
