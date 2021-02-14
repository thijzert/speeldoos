
if ( document.readyState == "loading" ) {
	document.addEventListener( "DOMContentLoaded", main );
} else {
	main();
}

import { streamPlayerMain } from "./components/stream-player.js";

import { homeMain } from "./pages/home.js";
import { statusMain } from "./pages/status.js";

function main() {
	streamPlayerMain();

	let ndMain = document.querySelector("main")
	if ( ndMain ) {
		let c = ndMain.classList;
		if ( c.contains("home") ) {
			homeMain();
		} else if ( c.contains("status") ) {
			statusMain();
		}
	}
}
