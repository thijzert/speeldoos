export class StreamPlayer {
	constructor(elt, streams) {
		let self = this;

		this.elt = elt;

		this.audio = new Audio();
		this.audio.controls = false;
		this.audio.preload = "none";

		this.streams = [];
		this.audioIndex = -1;
		this.playing = false;
		this.muted = false;

		elt.classList.add("stream-player")
		elt.innerHTML = "";

		let top = div(elt, "-top");
		let bot = div(elt, "-bottom");

		this.playpause = div(top, "-playpause");
		this.streamSelect = div(top, "-stream-select");

		this.volumeKnob = document.createElement("INPUT");
		this.volumeKnob.type = "range";
		this.volumeKnob.step = 0.25;
		this.volumeKnob.min = 0;
		this.volumeKnob.max = 100;
		this.volumeKnob.value = 100;
		this.volumeKnob.onchange = () => { self.setVolume(self.volumeKnob.value); }
		this.volumeKnob.oninput = () => { self.setVolume(self.volumeKnob.value); }
		div(bot, "-volume-knob").appendChild(this.volumeKnob);

		let i = 0;
		for ( let streamName in streams ) {
			if ( !streams.hasOwnProperty(streamName) ) {
				continue;
			}

			let audioIndex = i;
			this.streams[audioIndex] = new URL(streams[streamName], document.baseURI);
			i++;

			let btn = div(this.streamSelect, "-src");
			btn.onclick = () => {
				self.setAudioIndex(audioIndex);
			}
			btn.innerText = streamName;
		}

		if ( i > 0 ) {
			this.streamSelect.childNodes.forEach((x) => {
				widthP(x, 1/i);
			});
		}

		this.setAudioIndex(0);
		this.playpause.onclick = () => {
			if ( self.playing ) {
				self.stop();
			} else {
				self.play();
			}
		}

		this.audio.onended = () => {
			self.updateSrc();
		}
		this.audio.onstalled = () => {
			console.log("stalled");
			// self.updateSrc();
		}
		this.audio.onwaiting = () => {
			console.log("waiting");
			// self.updateSrc();
		}
	}

	setVolume(volume) {
		if ( volume < 0 ) {
			volume = 0;
		} else if ( volume > 100 ) {
			volume = 100;
		}

		if ( this.volumeKnob.value != volume ) {
			let c = this.volumeKnob;
			window.setTimeout( () => {
				c.value = volume;
			}, 20 );
		}

		// Setting the actual volume to x⁴ allows finer control at the quiet end
		let v = volume / 100;
		v = v*v*v*v;

		this.audio.volume = v;
	}

	setAudioIndex(audioIndex) {
		if ( this.audioIndex == audioIndex ) {
			return;
		}

		let cur = this.streamSelect.children.item(this.audioIndex);
		if ( cur ) {
			cur.classList.remove("-active");
		}
		this.audioIndex = audioIndex;
		cur = this.streamSelect.children.item(this.audioIndex);
		if ( cur ) {
			cur.classList.add("-active");
		}

		if ( this.playing ) {
			this.audio.pause();
			this.audio.play();
		}
		this.updateSrc();
	}

	updateSrc() {
		if ( this.playing ) {
			this.audio.pause();
		}

		let u = new URL(this.streams[this.audioIndex]);
		u.searchParams.set("_", (Math.random()).toString().substr(2));
		this.audio.src = u;

		if ( this.playing ) {
			this.audio.play();
		}
	}

	async play() {
		this.playing = true;
		this.playpause.classList.add("-playing");
		this.audio.play();
	}

	async stop() {
		this.playing = false;
		this.playpause.classList.remove("-playing");
		this.audio.pause();
	}
}

export function streamPlayerMain() {
	document.querySelectorAll(".-js-create-stream-player").forEach((x) => {
		x.classList.remove("-js-create-stream-player");

		let streams = {
			"HQ": "stream.mp3",
			"DOG": "stream.wav",
		};

		// TODO: get the list of streams from a data attribute

		let sp = new StreamPlayer(x, streams);
	} );
}

function div(parentNode, cssClass) {
	let rv = document.createElement("DIV");
	rv.classList.add(cssClass);
	parentNode.appendChild(rv);
	return rv;
}

function widthP(elt, frac) {
	if ( elt && elt.style ) {
		elt.style.width = (100 * frac).toString() + "%";
	}
}
