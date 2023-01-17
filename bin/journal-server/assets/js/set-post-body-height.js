(() => {
	for ( let autogrower of document.querySelectorAll(".auto-grow-textarea") ) {
		const textarea = autogrower.querySelector("textarea")
		if ( !textarea ) {
			return;
		}
		const afterInput = document.createElement("A");
		autogrower.parentElement.insertBefore(afterInput, autogrower.nextSibling);

		textarea.addEventListener("input", (e) => {
			autogrower.dataset.replicatedValue = textarea.value;

			if ( textarea.selectionEnd && textarea.selectionEnd > textarea.value.length-20 ) {
				// Cursor is at the end of input. Have some scrolloff space at the bottom of the screen
				let cr = afterInput.getClientRects()[0];

				if ( cr.y < 0 || cr.y > window.innerHeight ) {
					afterInput.scrollIntoView(false);
				}
			}
		});
	}
})()
