(() => {
	for ( let autogrower of document.querySelectorAll(".auto-grow-textarea") ) {
		const textarea = autogrower.querySelector("textarea")
		if ( !textarea ) {
			return;
		}

		textarea.addEventListener("input", (e) => {
			autogrower.dataset.replicatedValue = textarea.value;
		});
	}
})()
