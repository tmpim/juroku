os.loadAPI("juroku.lua")

local JUROKU_HOST = ""

local standby1 = require("standby1")
local standby2 = require("standby2")
local standby3 = require("standby3")
local standby4 = require("standby4")

if not debug then error("Missing debug API") end
local dp
for i = 1, 16 do
  local name, value = debug.getupvalue(peripheral.getNames, i)
  if name == "native" then
    dp = value
    break
  end
end
if not dp then error("failed to get direct peripheral access") end

local function wrapRemote(id, side)
  local call = dp.call
  return function(method, ...)
    return call(side, "callRemote", id, method, ...)
  end
end

local monitors = {
  wrapRemote("monitor_576", "bottom"),
  wrapRemote("monitor_577", "bottom"),
  wrapRemote("monitor_578", "bottom"),
  wrapRemote("monitor_579", "bottom")
}

local monitorsNormal = {
  peripheral.wrap("monitor_576"),
  peripheral.wrap("monitor_577"),
  peripheral.wrap("monitor_578"),
  peripheral.wrap("monitor_579")
}

local statusMonitor = monitorsNormal[1]
local debugMonitor = peripheral.wrap("top")

local driveA = peripheral.wrap("tape_drive_54")
local driveB = peripheral.wrap("tape_drive_55")

if driveA == nil then
  printError("Cannot find peripherals, network disconnected or server just started?")
  printError("Rebooting in 5 secs")
  sleep(5)
  os.reboot()
end

local function clearMonitors()
  for k, m in pairs(monitors) do
    local w, h = m("getSize")
    m("setPaletteColour", colours.black, 0x000000)
    m("setBackgroundColour", colours.black)
    m("clear")
    m("setTextScale", 0.5)
    m("setCursorPos", 1, 1)
  end
end

local function writeStatus(text, colour)
  statusMonitor.setPaletteColour(colours.black, 0x000000)
  statusMonitor.setPaletteColour(colours.white, colour)
  statusMonitor.setBackgroundColour(colours.black)
  statusMonitor.setTextColour(colours.white)
  statusMonitor.setTextScale(2)
  statusMonitor.clear()

  statusMonitor.setCursorPos(3, 2)
  statusMonitor.write(text)
end

local function drawStandby()
  clearMonitors()
  for k, m in pairs(monitorsNormal) do
    m.setTextScale(0.5)
  end
  standby1.draw(monitorsNormal[1])
  standby2.draw(monitorsNormal[2])
  standby3.draw(monitorsNormal[3])
  standby4.draw(monitorsNormal[4])
end

local function run()
  clearMonitors()
  print("Connecting")

  local ok, endpoint = http.websocketAsync(JUROKU_HOST)
  if not ok then error("couldnt connect") end
  local ws
  local decoder = juroku.Decoder.new(monitors, driveA, driveB, debugMonitor)

  parallel.waitForAll(function()
    while true do
      local e = {os.pullEvent()}
      local event = e[1]

      if event == "websocket_success" then
        ws = e[3]
        print("Connected!")
      elseif event == "websocket_failure" then
        error("Connection failed")
      elseif event == "websocket_message" then
        decoder:handleMessage(e[3])
      end
    end
  end, function()
    decoder:playVideo()
  end)

  if ws then ws.close() end
end

while true do
  driveA.stop(); driveB.stop()
  clearMonitors()

  -- print the dimensions of the cinema screen
  term.setTextColour(colours.grey)
  local tw = 0
  local th = 0
  statusMonitor.setTextScale(0.5)
  for k, m in pairs(monitors) do
    local w, h = m("getSize")
    tw = tw + w
    th = th + h
    print(k, w, h, w * 2, h * 3)
  end

  --writeStatus("No video playing", 0x555555)
  --drawStandby()

  term.setTextColour(colours.white)
  print("Press enter to connect")
  read()

  term.clear()
  term.setCursorPos(1, 1)
  term.setTextColour(colours.yellow)
  local ok, err = pcall(run, file)

  driveA.stop(); driveB.stop()
  clearMonitors()

  if not ok then
    if err == "Terminated" then
      writeStatus("Manually terminated", 0xFF0000)
      printError("Terminated")
    else
      writeStatus("Crashed", 0xFF0000)

      printError("Playback crashed: ")
      printError(err)
    end
    printError(" ")
    printError("Press enter to continue")
    read()
  end
end
