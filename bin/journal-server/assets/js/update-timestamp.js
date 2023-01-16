
let zp = function(i) {
	if ( i < 10 ) {
		return "0" + i.toString();
	}
	return i.toString();
}
let upd_ts = function() {
	let ipt_ts = document.getElementById("ipt-ts");
	if ( !ipt_ts ) {
		return;
	}
	let d = new Date();
	ipt_ts.placeholder = d.getFullYear() + "-" + zp(d.getMonth() + 1) + "-" + zp(d.getDate()) + " " + zp(d.getHours()) + ":" + zp(d.getMinutes());
};
upd_ts();
window.setInterval(upd_ts, 5000);

