
import { reloadBufferStatus } from "../components/buffer-status.js";

async function reloadNowPlaying() {
	let ndnp = document.querySelectorAll(".-js-load-now-playing");
	if ( ndnp.length == 0 ) {
		return;
	}

	let np = await fetch("/now-playing");
	let npb = await np.text();

	ndnp.forEach(nd => { nd.innerHTML = npb; });
}

export function statusMain() {
	window.setInterval(reloadNowPlaying, 4000);
	reloadNowPlaying();

	window.setInterval(reloadBufferStatus, 700);
	reloadBufferStatus();
}
