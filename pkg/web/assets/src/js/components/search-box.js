
export class SearchBox {
	constructor(elt) {
		let self = this;
		this.elt = elt;

		this._searchInput = this._must(".search-input input.-search-bar");

		this._results = this._must(".search-results .-results");
		this._noresults = this._must(".search-results .-no-results");
		this._resultCounter = this._must(".search-results .-result-counter");

		this._resultCounter.innerHTML = "<strong>1</strong>-<strong>3</strong> <span>/</span> <strong>71</strong>";

		this._lastSearch = "";

		this._searchInput.addEventListener("keyup", () => { self.updateSearchResults(); });
		this._searchInput.addEventListener("change", () => { self.updateSearchResults(); });
		this._searchInput.addEventListener("blur", () => { self.updateSearchResults(); });

		this.updateSearchResults();
	}

	_must(selector) {
		let rv = this.elt.querySelector(selector);
		if ( !rv ) {
			throw "Selector '" + selector + "' not found";
		}
		return rv;
	}

	async updateSearchResults() {
		let q = this._searchInput.value || "";
		if ( q === this._lastSearch ) {
			return;
		}

		this._lastSearch = q;

		if ( q === "" ) {
			this._noresults.style.display = "none";
			this._results.innerHTML = "";
			this._resultCounter.style.display = "none";
			return;
		}

		let u = new URL("/api/search", document.baseURI);
		u.searchParams.set("q", q);

		u.searchParams.set("_", (Math.random()).toString().substr(2));
		let resp = await fetch(u);
		let results = await resp.text();

		this._results.innerHTML = results;

		let nresults = this._results.querySelectorAll(".result");
		if ( nresults.length === 0 ) {
			this._noresults.style.display = "block";
			this._resultCounter.style.display = "none";
		} else {
			this._resultCounter.innerHTML = `<strong>1</strong> <span>/</span> <strong>${nresults.length}</strong>`;
			this._noresults.style.display = "none";
			this._resultCounter.style.display = "block";
		}

		nresults.forEach((nd) => {
			nd.querySelectorAll(".-performance").forEach((ndpf) => {
				if ( !ndpf.id ) {
					return;
				}

				let p = document.createElement("DIV");
				p.classList.add("-add-queue");
				let btn = document.createElement("BUTTON");
				btn.setAttribute("class", "tfbutton");
				btn.onclick = btnAddToQueue(btn, ndpf.id);
				btn.innerText = "add to queue"; // TODO: i18n
				p.appendChild(btn);
				ndpf.appendChild(p);
			});
		});
	}
}

function btnAddToQueue(btn, pfid) {
	return (async (e) => {
		btn.disabled = true;
		console.log("Add ", pfid, " to queue");

		let u = new URL("/api/queue/add", document.baseURI);
		var fd = new FormData()
		fd.append( "id", pfid );
		let resp = await fetch(u, {
			method: "POST",
			body: fd,
		});
		await resp.json();

		btn.disabled = false;
	});
}

export function searchBoxMain() {
	document.querySelectorAll(".search").forEach((x) => {
		let sb = new SearchBox(x);
	} );
}


