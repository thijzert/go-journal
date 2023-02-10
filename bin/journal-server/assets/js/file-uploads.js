(() => {
	const file_ipt = document.getElementById("ipt-file-upload");
	const file_list = document.getElementById("list-of-attached-files");

	if ( !file_ipt || !file_ipt.files || !file_list ) {
		return;
	}

	const CHUNKSIZE = 125000;

	let attachments = {};

	const upload_buf = async (hash) => {
		console.log(hash, attachments[hash]);
		const file = attachments[hash];
		if ( !file ) {
			return;
		}

		let chunk = file.buf.slice(file.offset, file.offset+CHUNKSIZE);
		if ( chunk.byteLength == 0 ) {
			file.pr.parentNode.appendChild(document.createTextNode("√"));
			file.pr.remove();
			return;
		}

		let this_url = new URL(location.href);
		let att_url = new URL("journal/attachment", location.href);
		for ( let k of this_url.searchParams.keys() ) {
			att_url.searchParams.set(k, this_url.searchParams.get(k));
		}
		att_url.searchParams.set("att_hash", file.hash);

		let q = await fetch(att_url, {method: "POST", body: chunk});
		q = await q.json();
		if ( !q.ok ) {
			console.error(q);
			file.pr.parentNode.appendChild(document.createTextNode("×"));
			file.pr.remove();
			return;
		}

		file.pr.max = Math.ceil(file.buf.byteLength / CHUNKSIZE);
		file.pr.value = Math.floor(file.offset / CHUNKSIZE);
		file.offset += chunk.byteLength;

		return await upload_buf(hash);
	}

	file_ipt.addEventListener("input", async (e) => {
		let files = [];
		for ( let f of file_ipt.files ) {
			files.push(f);
		}
		file_ipt.files = null;

		let headstart = 0;
		for ( let f of files ) {
			let buf = await f.arrayBuffer();
			let hashBuffer = await crypto.subtle.digest('SHA-256', buf);
			let hash = Array.from(new Uint8Array(hashBuffer)).map((b) => b.toString(16).padStart(2, '0')).join('');

			if ( hash in attachments ) {
				continue;
			}

			let li = document.createElement("li");
			let lbl = document.createElement("label");
			lbl.textContent = f.name;
			let check = document.createElement("input");
			check.type = "checkbox";
			check.name = "attachment-"+hash;
			check.checked = true;
			lbl.insertBefore(check, lbl.firstChild);
			li.appendChild(lbl);

			let pr = document.createElement("progress");
			li.appendChild(pr);

			file_list.appendChild(li);
			let offset = 0;
			attachments[hash] = {hash, buf, pr, li, offset};
			window.setTimeout(() => { upload_buf(hash); }, headstart);
			headstart += 200;
		}
	})
})();
