package main

var audioHTML = `
<!DOCTYPE html>
<html>
<head>
	<title>juroku/retro: audio system</title>
	<link href="https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap" rel="stylesheet">
	<script type="text/javascript" src="https://code.jquery.com/jquery-3.3.1.slim.min.js"></script>
	<!-- <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/fomantic-ui@2.8.4/dist/semantic.min.css"> -->
	<!-- <script src="https://cdn.jsdelivr.net/npm/fomantic-ui@2.8.4/dist/semantic.min.js"></script> -->
	<style>
	body {
		text-align: center;
		font-family: 'Press Start 2P', 'Arial', sans-serif;
	}
	</style>
</head>
<body>
	<h1>welcome to juroku/retro's audio system</h1>
	<p>we hope you enjoy your stay</p>
	<noscript><p>P.S. you will need javascript enabled to hear the audio</p></noscript>

	<button onclick="connect()">Connect</button>
<script type="text/javascript">

const bufferLen = 500000;
let leftBuffer = new Int16Array(bufferLen);
let rightBuffer = new Int16Array(bufferLen);
let head = 0;
let tail = 0;
let targetDelay = 4096;
let currentTarget = targetDelay;
let maxDelay = 8192;

let audioCtx = null;
let audioReader = null;

let start = (sampleRate) => {
	audioCtx = new AudioContext({
		sampleRate: sampleRate
	})
	audioReader = audioCtx.createScriptProcessor(sampleRate / 64, 1, 2)

	audioReader.onaudioprocess = event => {
		let targetDelta = currentTarget - head;
		targetDelta = (targetDelta < 0 && targetDelta < -bufferLen/2) ? bufferLen + targetDelta : targetDelta;

		let outL = event.outputBuffer.getChannelData(0);
		let outR = event.outputBuffer.getChannelData(1);

		if (currentTarget > 0) {
			console.log("target delta:", targetDelta, head - tail);
		}

		if (currentTarget > 0 && targetDelta > 0) {
			console.log("waiting for target head:", targetDelta);
			for (let i = 0; i < outL.length; i++) {
				outL[i] = 0;
				outR[i] = 0;
			}
			return;
		}

		currentTarget = -1;

		let delta = head - tail;
		delta = (delta < 0 && delta < -bufferLen/2) ? bufferLen + delta : delta;

		if (delta >= maxDelay) {
			tail += (delta - targetDelay);
			console.log("falling behind, jumped tail by:", delta - targetDelay);
			console.log("new state:", head-tail, tail, head);
		}

		for (let i = 0; i < outL.length; i++) {
			if (tail == head) {
				outL[i] = 0;
				outR[i] = 0;
				continue;
			}

			outL[i] = leftBuffer[tail] / 32767;
			outR[i] = rightBuffer[tail] / 32767;

			// outL[i] = ((Math.random() * 2) - 1) * 0.1;
			// outR[i] = ((Math.random() * 2) - 1) * 0.1;

			tail = (tail + 1) % bufferLen;
			if (tail == head) {
				// uh oh, we ran out of buffer
				console.log('warning: out of buffer, buffering...');
				currentTarget = (head + targetDelay) % bufferLen;
			}
		}
	}

	audioReader.connect(audioCtx.destination);
}

let connect = () => {
	const ws = new WebSocket((window.location.protocol == 'https:' ? 'wss://' : 'ws://') + window.location.host + '/api/ws/audio');
	ws.binaryType = 'arraybuffer';

	ws.addEventListener('open', event => {
		console.log("good news, we successfully connected to the server");
	});

	ws.addEventListener('message', event => {
		if (audioCtx == null) {
			console.log("sample rate:", event.data);
			start(parseFloat(event.data));
			return;
		}
		const rawData = new Int16Array(event.data);
		for (let i = 0; i < rawData.length; i += 2) {
			leftBuffer[head] = rawData[i];
			rightBuffer[head] = rawData[i+1];
			head = (head + 1) % bufferLen;
		}
	});
}

</script>
</body>
</html>
`
