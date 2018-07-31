local hex = {"0","1","2","3","4","5","6","7","8","9","a","b","c","d","e","f"}

Decoder = {}

local typeImage = 1
local typeVideo = 2
local typeVideoNoAudio = 3
local typeAudio = 4

local frameRate = 10
local dataRate = 48000 / 8
local startDelay = 1.25 * dataRate

function Decoder.new(monitors, file, driveA, driveB)
	local self = {monitors = monitors, file = file, frame = 0, driveA = driveA,
		driveB = driveB, writingDrive = driveA, playingDrive = driveA,
		audioBuffer = {}, transDuration = 3, transitionBuffer = {},
		loadedRows = {}, loadedColors = {}}
	local magic = string.char(file.read()) .. string.char(file.read()) .. string.char(file.read())
	if magic ~= "JUF" then
		error("juroku: not a valid JUF file")
	end

	local version = file.read()
	if version ~= 1 then
		error("juroku: JUF file version " .. version .. " not supported")
	end

	self.type = file.read()

	if self.type ~= typeImage and self.type ~= typeVideo then
		error("juroku: this decoder does not support this JUF media type")
	end

	if self.type == typeVideo and ((not driveA) or (not driveB)) then
		error("juroku: two tape drives as arguments required for audio output")
	end

	local numMonitors = file.read()
	if numMonitors ~= #monitors then
		-- error("juroku: file requires " .. numMonitors ..
			-- " monitors, but only " .. #monitors .. " given")
	end

	for i = 1, numMonitors do
		self.loadedRows[i] = {}
		self.loadedColors[i] = {}
	end

	setmetatable(self, {__index = Decoder})
	return self
end

function Decoder:read(size)
	if size <= 0 then
		return ""
	elseif size < 16 * 1023 then
		return self.file.read(size)
	elseif size then
		local result = ""
		while true do
			local nextRead = 16 * 1023
			if size < nextRead then
				nextRead = size
			end

			result = result .. self.file.read(nextRead)
			size = size - nextRead

			if size == 0 then
				return result
			end
		end
	else
		return self.file.read()
	end
end

function Decoder:hasAudio()
	return self.type == typeVideo or self.type == typeAudio
end

function Decoder:parseAudio(shouldBuffer)
	local f = self.file
	local sizeArr = f.read(4)
	local size = sizeArr:byte(1) * 0x1000000 + sizeArr:byte(2) * 0x10000 + sizeArr:byte(3) * 0x100 + sizeArr:byte(4)
	local data = self:read(size)

	if self.writingDrive == nil then
		-- if shouldBuffer then
			for i = 1, size do
				table.insert(self.transitionBuffer, data:byte(i))
			end
		-- else
		-- 	for i = 1, size do
		-- 		local result = f.read()
		-- 		table.insert(self.audioBuffer, result)
		-- 	end
		-- end
		return
	end

	-- if #self.audioBuffer > 0 then
	-- 	for i = 1, #self.audioBuffer do
	-- 		self.writingDrive.write(self.audioBuffer[i])
	-- 	end
	-- 	self.audioBuffer = {}
	-- end

	-- if shouldBuffer then
	-- 	table.concat(self.transitionBuffer, data)
	-- end

	for i = 1, size do
		local b = data:byte(i)
		self.writingDrive.write(b)
		table.insert(self.transitionBuffer, b)

		if i % (60 * dataRate) == 0 then
			os.queueEvent("juroku_audio_yield")
			coroutine.yield()
		end
	end
end

function Decoder:loadFrame(frame, shouldBuffer)
	local f = self.file

	if #self.loadedColors > 0 then
		self.loadedColors[1] = {}
	end

	if frame < 0 then
		return true
	end

	-- print("skipping...")
	for i = self.frame, frame - 1 do
		for m = 1, #self.monitors do
			local first = f.read()
			if first == nil then
				return false
			end

			local width = first * 0x100 + f.read()
			local height = f.read() * 0x100 + f.read()

			self:read(width * (height * 3) + (16 * 3))
		end

		if self:hasAudio() then
			self:parseAudio(shouldBuffer)
		end
	end

	-- print("reading...")
	for m, t in pairs(self.monitors) do
		local first = f.read()
		if first == nil then
			return false
		end

		local width = first * 0x100 + f.read()
		local height = f.read() * 0x100 + f.read()
		local monitorRows = self.loadedRows[m]
		local monitorColors = self.loadedColors[m]

		-- local image = self:read(width * (height * 2))

		for row = 1, height do
			-- t.setCursorPos(1, row)
			-- Use file.read here to improve performance
			-- t.blit(f.read(width), f.read(width), f.read(width))
			-- local fg = ""
			-- local bg = ""
			-- local txt = ""
			-- for col = 1, width do
			-- 	local pos = ((row - 1) * width + col) * 2 - 1
			-- 	local color = string.byte(image:sub(pos, pos))
			-- 	fg = fg .. hex[math.floor(color / 0x10) + 1]
			-- 	bg = bg .. hex[bit.band(color, 0xF) + 1]
			-- 	txt = txt .. image:sub(pos+1, pos+1)
			-- end

			-- local text = f.read(width)
			-- local cols = f.read(width)

			-- local fg = ""
			-- local bg = ""
			-- for i = 1, width do
			-- 	local color = cols:byte(i)
			-- 	fg = fg .. hex[math.floor(color / 0x10) + 1]
			-- 	bg = bg .. hex[bit.band(color, 0xF) + 1]
			-- 	-- textColors = textColors .. cols[0]
			-- end

			monitorRows[row] = {f.read(width), f.read(width), f.read(width)}
		end

		local paletteData = f.read(16 * 3)

		for i = 1, 16 do
			local offset = (3 * i)
			monitorColors[i] = paletteData:byte(offset - 2) * 0x10000 +
				paletteData:byte(offset - 1) * 0x100 + paletteData:byte(offset)
		end
	end

	-- print("writing audio...")
	if self:hasAudio() then
		self:parseAudio(shouldBuffer)
	end

	-- print("done!")

	return true
end

function Decoder:drawFrame()
	if #self.loadedColors == 0 or #self.loadedColors[1] == 0 then
		return
	end

	-- os.queueEvent("juroku_frame")
	-- coroutine.yield()

	for m, t in pairs(self.monitors) do
		local monitorColors = self.loadedColors[m]
		for i = 1, 16 do
			t("setPaletteColor", 2^(i-1), monitorColors[i])
		end

		for row, data in pairs(self.loadedRows[m]) do
			t("setCursorPos", 1, row)
			t("blit", data[1], data[2], data[3])
		end
	end

	-- os.queueEvent("juroku_frame")
	-- coroutine.yield()
end

function Decoder:writeTransitionBuffer()
	if #self.transitionBuffer < startDelay then
		return
	end

	for i = #self.transitionBuffer - startDelay + 1, #self.transitionBuffer do
		self.writingDrive.write(self.transitionBuffer[i])
	end

	self.transitionBuffer = {}
end

function Decoder:playVideo()
	self.driveA.setSpeed(1)
	self.driveB.setSpeed(1)
	self.driveA.stop()
	self.driveB.stop()
	self.driveA.seek(-self.driveA.getSize())
	self.driveB.seek(-self.driveB.getSize())
	self:parseAudio(true)
	for i = 1, 3000 do
		self.driveA.write(0xAA)
	end
	self.driveA.seek(-self.driveA.getSize())

	local bufferLength = #self.transitionBuffer * dataRate
	local tapeEndThreshold = #self.transitionBuffer - startDelay
	local tapeEndCanary = #self.transitionBuffer - (startDelay * 2)
	sleep(0)

	self.writingDrive = self.driveB
	self:writeTransitionBuffer()

	self.playingDrive.play()
	local tapeOffset = -startDelay
	local currentFrame = -1
	local nextPlaying = nil
	local endTransition = 0
	local lastPos = -1
	local targetFrame = -1
	local nextOffset = 0
	local interpolatePos = 0

	while true do
		local playPos = self.playingDrive.getPosition()

		local totalSamples = tapeOffset + playPos

		if playPos ~= lastPos then
			targetFrame = math.floor((totalSamples / dataRate) * frameRate + 0.5)
			interpolatePos = playPos
			print(tapeOffset .. " + " .. playPos .. " = " .. totalSamples .. " | frame: " .. targetFrame)
			lastPos = playPos
		else
			targetFrame = targetFrame + 1
			interpolatePos = interpolatePos + (dataRate / frameRate)
		end

		if not self:loadFrame(targetFrame - currentFrame - 1, false) then
			self.playingDrive.stop()
			return
		end

		if targetFrame > currentFrame then
			currentFrame = targetFrame
		end

		if interpolatePos >= tapeEndThreshold then
			if nextPlaying == nil and self.writingDrive.getPosition() > startDelay then
				-- Write 0.5 seconds of silence
				for i = 1, 3000 do
					self.writingDrive.write(0xAA)
				end

				nextPlaying = self.writingDrive
				self.writingDrive = nil

				os.queueEvent("juroku_audio_yield")
				coroutine.yield()

				nextPlaying.seek(-nextPlaying.getSize())
				nextPlaying.play()
				print("playing transition")
				endTransition = tapeEndThreshold + startDelay
				nextOffset = interpolatePos
			elseif nextPlaying ~= nil and interpolatePos >= endTransition then
				print("transitioning...")

				os.queueEvent("juroku_audio_yield")
				coroutine.yield()

				self.playingDrive.stop()
				self.playingDrive.seek(-self.playingDrive.getSize())
				self.writingDrive = self.playingDrive
				self:writeTransitionBuffer()
				self.playingDrive = nextPlaying
				print("done!")
				endTransition = 0
				nextPlaying = nil
				tapeOffset = tapeOffset + nextOffset
				playPos = self.playingDrive.getPosition()
			end
		end


		-- os.queueEvent("juroku_frame")
		-- coroutine.yield()


		-- print("drawing frame")
		sleep(0.1)
		self:drawFrame()
		-- sleep(0)

		-- os.queueEvent("juroku_frame")
		-- coroutine.yield()
		-- sleep(0.2 - (os.clock() - start))
	end
end

function Decoder:render()
	if self.type == typeImage then
		drawFrame(0)
		return
	elseif self.type == typeVideo then
		self:playVideo()
	end
end
