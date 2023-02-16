const AUTOSAVE_INTERVAL_MS = 2500;
(async () => {
	const editform = document.querySelector("form.-js-autosave-draft");
	if ( !editform ) {
		return;
	}

	const ipt_body = editform.querySelector("textarea");
	const ipt_project = editform.querySelector("#ipt-project") || document.createElement("input");
	const ipt_draft_id = editform.querySelector("input[type=hidden][name=draft_id]");
	if ( !ipt_body || !ipt_draft_id ) {
		return;
	}

	const save_status = document.createElement("DIV");
	const save_time = document.createElement("DIV");

	let indicator = document.querySelector(".indicator-tray .save-status");
	if ( indicator ) {
		indicator.innerHTML = '';
		indicator.appendChild(save_status);
		indicator.appendChild(save_time);
	}

	let this_url = new URL(location.href);
	let draft_url = new URL("journal/draft", location.href);
	for ( let k of this_url.searchParams.keys() ) {
		draft_url.searchParams.set(k, this_url.searchParams.get(k));
	}

	let draft_id = "";
	let save_draft = async () => {
		save_status.innerText = "Autosaving...";
		try {
			let pb = new FormData();
			pb.set("draft_id", draft_id);
			pb.set("body", ipt_body.value);
			pb.set("project", ipt_project.value);
			let draft_emptied = (ipt_body.value.trim() === '');
			let q = await fetch(draft_url, {method: "POST", body: pb});
			q = await q.json();
			if ( q.ok == 1 ) {
				save_status.innerText = "Draft saved";
				save_time.innerText = `Last saved at ${(new Date()).toTimeString().slice(0,5)}`;
				draft_id = q.draft_id;

				if ( draft_emptied ) {
					save_status.innerText = "Draft deleted";
					save_time.innerText = "";
				}
			} else {
				save_status.innerText = "Error saving";
				console.error("unable to save draft", q);
			}
		} catch (e) {
			save_status.innerText = "\u274c Error saving";
			console.error(e);
			throw e;
		}
	};

	let current_body = ipt_body.value;
	let current_project = ipt_project.value;
	window.setInterval(async () => {
		if ( ipt_body.value == current_body && ipt_project.value == current_project ) {
			return;
		}

		try {
			await save_draft();
			current_body = ipt_body.value;
			current_project = ipt_project.value;
		} catch ( e ) {
			console.error("automatic save failed")
		}
	}, AUTOSAVE_INTERVAL_MS);
})();
