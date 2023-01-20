const AUTOSAVE_INTERVAL_MS = 2500;
(async () => {
	const editform = document.querySelector("form.-js-autosave-draft");
	if ( !editform ) {
		return;
	}

	const ipt_body = editform.querySelector("textarea");
	const ipt_draft_id = editform.querySelector("input[type=hidden][name=draft_id]");
	if ( !ipt_body || !ipt_draft_id ) {
		return;
	}

	let this_url = new URL(location.href);
	let draft_url = new URL("journal/draft", location.href);
	for ( let k of this_url.searchParams.keys() ) {
		draft_url.searchParams.set(k, this_url.searchParams.get(k));
	}

	let draft_id = "";
	let save_draft = async () => {
		try {
			let pb = new FormData();
			pb.set("draft_id", draft_id);
			pb.set("body", ipt_body.value);
			let q = await fetch(draft_url, {method: "POST", body: pb});
			q = await q.json();
			if ( q.ok == 1 ) {
				console.log("draft saved", q);
				draft_id = q.draft_id;
			} else {
				console.error("unable to save draft", q);
			}
		} catch (e) {
			console.error(e);
			throw e;
		}
	};

	let current_body = ipt_body.value;
	window.setInterval(async () => {
		if ( ipt_body.value == current_body ) {
			return;
		}

		try {
		await save_draft();
		current_body = ipt_body.value;
		} catch ( e ) {
			console.error("automatic save failed")
		}
	}, AUTOSAVE_INTERVAL_MS);
})();
