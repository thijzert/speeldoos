
async function reloadNowPlaying() {
	let ndnp = document.querySelectorAll(".-js-load-now-playing");
	if ( ndnp.length == 0 ) {
		return;
	}

	let np = await fetch("/now-playing");
	let npb = await np.text();

	ndnp.forEach(nd => { nd.innerHTML = npb; });
}

export function homeMain() {
	window.setInterval(reloadNowPlaying, 4000);
	reloadNowPlaying();
}
