--os.loadAPI("jurokudebug.lua")

local hex = {"0","1","2","3","4","5","6","7","8","9","a","b","c","d","e","f"}

Decoder = {}

local typeImage = 1
local typeVideo = 2
local typeVideoNoAudio = 3
local typeAudio = 4

local frameRate = 10
local dataRate = 48000 / 8
--local startDelay = 0.75 * dataRate
local delayFrames = 2
local initialBufferTime = 10
local missingFrameBufferThreshold = 10
local missingFrameBufferMultiplier = 2

local logFile, logStart

local function debugLog(text, colour)
  local col = term.getTextColour()
  term.setTextColour(colour or col)
  print(text)
  term.setTextColour(col)

  logFile.writeLine(string.format("[%8f] %s", (os.epoch("utc") - logStart) / 1000, text))
  logFile.flush()
end

local function byteStr(str)
  local out = ""

  for i = 1, #str do
    out = out .. str:byte(i) .. " "
  end

  return out:sub(1, #out - 1)
end

local function parseInt(str)
  return str:byte(1) * 0x1000000 + str:byte(2) * 0x10000 + str:byte(3) * 0x100 + str:byte(4)
end

local function parseShort(str)
  return str:byte(1) * 0x100 + str:byte(2)
end

function Decoder.new(monitors, driveA, driveB, debugMonitor)
  local self = {monitors = monitors, frame = 0, driveA = driveA,
    driveB = driveB, writingDrive = driveA, playingDrive = driveA,
    audioBuffer = {}, audioBufferN = 1, transDuration = 3, transitionBuffer = "",
    receivedFrames = {}, receivedFrameCounter = 0,
    loadedRows = {}, loadedColors = {},
    handledHeader = false, handledInitialFrame = false,
    readingChunk = -1, chunkData = "",
    debugMonitor = debugMonitor}

  if logFile ~= nil then logFile.close() end
  fs.delete("/.juroku.log")
  logFile = fs.open("/.juroku.log", "a")
  logStart = os.epoch("utc")

  --[[if debugMonitor ~= nil then
    jurokudebug.setMonitor(debugMonitor)
  end]]

  setmetatable(self, {__index = Decoder})
  return self
end

function Decoder:handleMessage(message)
  if #message == 0 then return end
  if #message == 4 and self.readingChunk == -1 then
    self.chunkData = ""
    self.readingChunk = parseInt(message:sub(1, 4))
    return
  end

  if self.readingChunk ~= -1 then
    self.chunkData = self.chunkData .. message
  else
    self.chunkData = message
  end

  if self.readingChunk == -1 or #self.chunkData >= self.readingChunk then
    if not self.handledHeader then
      self:handleHeader(self.chunkData)
      os.queueEvent("juroku_start")
      coroutine.yield()
    --[[elseif not self.handledInitialFrame then
      self:parseAudio(self.chunkData)
      self.handledInitialFrame = true
      os.queueEvent("juroku_start")
      coroutine.yield()]]
    else -- regular frame stream, hopefully
      --debugLog(string.format("Received frame %d   size %d", self.receivedFrameCounter, #self.chunkData))
      self.receivedFrames[self.receivedFrameCounter] = self.chunkData
      self.receivedFrameCounter = self.receivedFrameCounter + 1
    end

    self.readingChunk = -1
    self.chunkData = ""
  end
end

function Decoder:handleHeader(message)
  self.handledHeader = true

  if message:sub(1, 3) ~= "JUF" then
    error("juroku: not a valid JUF file")
  end

  local version = message:byte(4)
  if version ~= 1 then
    error("juroku: JUF file version " .. version .. " not supported")
  end

  self.type = message:byte(5)
  if self.type ~= typeVideo then
    error("juroku: this decoder does not support this JUF media type")
  end

  local numMonitors = message:byte(6)
  for i = 1, numMonitors do
    self.loadedRows[i] = {}
    self.loadedColors[i] = {}
  end
end

function Decoder:hasAudio()
  return self.type == typeVideo or self.type == typeAudio
end

function Decoder:parseAudio(data)
  --local length = parseInt(data:sub(1, 4))
  data = data:sub(5)

  --[[if self.writingDrive == nil then
    self.transitionBuffer = self.transitionBuffer .. data
    return
  end

  self.writingDrive.write(data)
  self.transitionBuffer = self.transitionBuffer .. data]]

  if self.audioBufferN >= 40 then
    self.driveA.seek(-self.driveA.getPosition())

    debugLog("A " .. #self.audioBuffer .. "  N " .. self.audioBufferN, colours.blue)

    for i = 1, self.audioBufferN - 1 do
      if not self.audioBuffer[i] then break end
      self.driveA.write(self.audioBuffer[i])
    end
    self.driveA.seek(-self.driveA.getPosition())

    self.audioBufferN = 1
  else
    self.audioBuffer[self.audioBufferN] = data
    self.audioBufferN = self.audioBufferN + 1
  end
end

function Decoder:loadFrame(lastFrame, frame)
  debugLog("Loading frame " .. frame, colours.green)

  if #self.loadedColors > 0 then
    self.loadedColors[1] = {}
  end

  if frame < 0 then
    return true
  end

  if frame - lastFrame > 1 then
    debugLog(string.format("Skipped %d frames", frame - lastFrame))
  end

  for i = lastFrame, frame - 1 do
    debugLog("Skipping frame " .. i)
    if self.receivedFrames[i] ~= nil then
      local data = self.receivedFrames[i]
      local bytei = 0

      if i ~= 0 then
        self.receivedFrames[i] = nil
      end

      for m = 1, #self.monitors do
        local wb = data:sub(bytei + 1, bytei + 2)
        local hb = data:sub(bytei + 3, bytei + 4)

        if wb == nil or hb == nil then -- EOF
          return true
        end

        local width = parseShort(wb)
        local height = parseShort(hb)

        if width * height == 0 then -- EOF
          return true
        end

        bytei = bytei + 4 + (width * (height * 3) + (16 * 3))
      end

      if self:hasAudio() then
        -- TODO: may need ± 1
        self:parseAudio(data:sub(bytei))
      end
    else
      debugLog("Skipped frame " .. i .. " doesn't exist", colours.red)
    end
  end

  local data = self.receivedFrames[frame]
  local bytei = 0

  if data == nil then
    debugLog("Frame " .. frame .. " doesn't exist", colours.red)

    return true
  end

  self.receivedFrames[frame] = nil

  for m, t in pairs(self.monitors) do
    local wb = data:sub(bytei + 1, bytei + 2)
    local hb = data:sub(bytei + 3, bytei + 4)

    if wb == nil or hb == nil then -- EOF
      return true
    end

    local width = parseShort(wb)
    local height = parseShort(hb)

    if width * height == 0 then -- EOF
      return true
    end

    local monitorRows = self.loadedRows[m]
    local monitorColors = self.loadedColors[m]

    bytei = bytei + 4

    for row = 1, height do
      monitorRows[row] = {
        data:sub(bytei +               1, bytei +  width     ),
        data:sub(bytei +  width      + 1, bytei + (width * 2)),
        data:sub(bytei + (width * 2) + 1, bytei + (width * 3)),
      }

      bytei = bytei + (width * 3)
    end

    local paletteData = data:sub(bytei + 1, bytei + (16 * 3))
    if paletteData == nil or #paletteData ~= 16 * 3 then -- EOF
      return true
    end

    bytei = bytei + 16 * 3

    for i = 1, 16 do
      local offset = (3 * i)
      monitorColors[i] = paletteData:byte(offset - 2) * 0x10000 +
        paletteData:byte(offset - 1) * 0x100 + paletteData:byte(offset)
    end
  end

  if self:hasAudio() then
    self:parseAudio(data:sub(bytei))
  end

  return true
end

function Decoder:drawFrame()
  if #self.loadedColors == 0 or #self.loadedColors[1] == 0 then
    return
  end

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
end

function Decoder:writeTransitionBuffer()
  if #self.transitionBuffer < startDelay then
    return
  end

  self.writingDrive.write(self.transitionBuffer:sub(#self.transitionBuffer - startDelay + 1, #self.transitionBuffer))

  self.transitionBuffer = ""
end

function Decoder:playVideo()
  --self.driveB.setSpeed(1)
  --self.driveB.stop()
  --self.driveB.seek(-self.driveB.getSize())

  print("Clearing tape")

  self.driveA.setSpeed(1)
  self.driveA.stop()
  self.driveA.seek(-self.driveA.getSize())
  self.driveA.write(string.char(0xAA):rep(self.driveA.getSize()))
  self.driveA.seek(-self.driveA.getSize())

  os.pullEvent("juroku_start")
  print("Starting playback in " .. initialBufferTime .. " seconds")
  sleep(initialBufferTime)

  --local bufferLength = #self.transitionBuffer * dataRate
  --local tapeEndThreshold = #self.transitionBuffer - startDelay
  --local tapeEndCanary = #self.transitionBuffer - (startDelay * 2)
  sleep(0)

  --self.writingDrive = self.driveB
  --self:writeTransitionBuffer()

  --self.playingDrive.play()
  self.driveA.play()
  --local tapeOffset = -startDelay
  local tapeOffset = 0
  local tapeCapacity = self.driveA.getSize()
  local currentFrame = -1
  local nextPlaying = nil
  --local endTransition = 0
  local lastPos = -1
  local targetFrame = -1
  local nextOffset = 0
  local interpolatePos = 0
  --local transitionedOnce = false

  while true do
    --[[local start = os.clock()
    local playPos = self.driveA.getPosition()
    local totalSamples = tapeOffset + playPos

    if playPos ~= lastPos then
      oldTargetFrame = targetFrame
      targetFrame = math.floor((totalSamples / dataRate) * frameRate + 0.5)
      interpolatePos = playPos
      lastPos = playPos
    else
      targetFrame = targetFrame + 1
      interpolatePos = interpolatePos + (dataRate / frameRate)
    end]]

    targetFrame = targetFrame + 1

    --jurokudebug.updateTapeInfo(self, interpolatePos, tapeEndThreshold)

    if not self:loadFrame(currentFrame + 1, targetFrame) then
      self.driveA.stop()
      return
    end

    if targetFrame > currentFrame then
      currentFrame = targetFrame
    end

    -- if we're very behind, wait for some new frames for a bit
    --[[if targetFrame - self.receivedFrameCounter > missingFrameBufferThreshold then
      self.driveA.stop()
      sleep(frameRate * missingFrameBufferThreshold * missingFrameBufferMultiplier)
      self.driveA.play()
    end]]

    --[[if interpolatePos >= tapeEndThreshold then
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
        endTransition = tapeEndThreshold + startDelay
        nextOffset = interpolatePos
      elseif nextPlaying ~= nil and interpolatePos >= endTransition then
        local col = term.getTextColour()
        term.setTextColour(colours.green)
        debugLog("transitioning...", colours.green)
        term.setTextColour(col)

        os.queueEvent("juroku_audio_yield")
        coroutine.yield()

        self.playingDrive.stop()
        self.playingDrive.seek(-self.playingDrive.getSize())
        self.writingDrive = self.playingDrive
        self:writeTransitionBuffer()
        self.playingDrive = nextPlaying
        endTransition = 0
        nextPlaying = nil
        tapeOffset = tapeOffset + nextOffset
        playPos = self.playingDrive.getPosition()
      end
    end]]

    sleep(0.09)

    self:drawFrame()
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
