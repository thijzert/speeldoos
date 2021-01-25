
export async function reloadBufferStatus() {
	let ndnp = document.querySelectorAll(".-js-load-buffer-status");
	if ( ndnp.length == 0 ) {
		return;
	}

	let np = await fetch("/api/status/buffers");
	let npb = await np.json();

	let tahead = 45000, tbehind = 90000;
	for ( let k in npb ) {
		tahead = Math.max(npb[k].Tahead, tahead);
		tbehind = Math.max(npb[k].Tbehind, tbehind);
	}

	let setWidth = (nd, t) => {
		nd.style.width = ((100 * t) / (tahead + tbehind)) + "%";
	};

	ndnp.forEach(nd => {
		nd.querySelectorAll(".-buffer").forEach(ndbuf => {
			let ndnow = ndbuf.querySelector(".-now");
			if ( !ndnow ) {
				ndnow = document.createElement("DIV");
				ndnow.classList.add("-now");
				ndbuf.appendChild(ndnow);
			}
			ndnow.style.left = ((100 * tbehind) / (tahead + tbehind)) + "%";

			if ( !ndbuf.dataset["buffer"] || !npb.hasOwnProperty(ndbuf.dataset["buffer"]) ) {
				return
			}
			let mybuf = npb[ndbuf.dataset["buffer"]];

			let sanity = ndbuf.querySelectorAll(".-offset-left, .-offset-right, .-the-rest .-bar .-past, .-the-rest .-bar .-future, .-the-rest .-labels .-past, .-the-rest .-labels .-future");
			if ( sanity.length != 6 ) {
				ndbuf.innerHTML = `<div class="-offset-left"></div>
					<div class="-the-rest">
						<div class="-bar"><div class="-past"></div><div class="-future"></div></div>
						<div class="-labels"><div class="-past"></div><div class="-future"></div></div>
					</div>
					<div class="-offset-right"></div>`;
			}

			let offset_l = ndbuf.querySelector(".-offset-left");
			let offset_r = ndbuf.querySelector(".-offset-right");
			let the_rest = ndbuf.querySelector(".-the-rest");

			let bar_p = ndbuf.querySelector(".-the-rest .-bar .-past");
			let bar_f = ndbuf.querySelector(".-the-rest .-bar .-future");
			let lbl_p = ndbuf.querySelector(".-the-rest .-labels .-past");
			let lbl_f = ndbuf.querySelector(".-the-rest .-labels .-future");

			if ( !offset_l || !offset_r || !bar_p || !bar_f || !lbl_p || !lbl_f ) {
				ndbuf.innerHTML = "(we'll get it on the next one)";
				return;
			}

			setWidth(offset_l, (tbehind - mybuf.Tbehind));
			setWidth(offset_r, (tahead - mybuf.Tahead));

			setWidth(the_rest, (mybuf.Tbehind + mybuf.Tahead));
			let localWidth = (nd, t) => {
				nd.style.width = ((100 * t) / (mybuf.Tbehind + mybuf.Tahead)) + "%";
			};

			localWidth(bar_p, Math.min(mybuf.Tbehind, mybuf.Tbehind + mybuf.Tahead))
			localWidth(lbl_p, Math.min(mybuf.Tbehind, mybuf.Tbehind + mybuf.Tahead))

			localWidth(bar_f, Math.max(mybuf.Tahead, 0))
			localWidth(lbl_f, Math.max(mybuf.Tahead, 0))

			lbl_p.innerText = (0.001*mybuf.Tbehind).toFixed(1);
			lbl_f.innerText = (0.001*mybuf.Tahead).toFixed(1);
		});
	});
}

