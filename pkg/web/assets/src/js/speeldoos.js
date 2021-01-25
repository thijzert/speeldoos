
if ( document.readyState == "loading" ) {
	document.addEventListener( "DOMContentLoaded", main );
} else {
	main();
}

import { homeMain } from "./pages/home.js";
import { statusMain } from "./pages/status.js";

console.log("boot POST");

function main() {
	let ndMain = document.querySelector("main")
	console.log("main", ndMain);
	if ( ndMain ) {
		let c = ndMain.classList;
		if ( c.contains("home") ) {
			homeMain();
		} else if ( c.contains("status") ) {
			statusMain();
		}
	}
}
