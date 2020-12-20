
local Decoder = require("jurokunext")

-- local JUROKU_HOST = "ws://me.lemmmy.pw:8089/stream"
-- local JUROKU_HOST = "ws://switchcraft.pw:9999/api/client"
-- local JUROKU_VIDEO = "ws://bf677dd2.eu.ngrok.io/api/ws/video"
-- local JUROKU_CONTROL = "ws://bf677dd2.eu.ngrok.io/api/ws/control/0"
local JUROKU_VIDEO = "ws://quincy.chuie.io:9999/api/ws/video"
local JUROKU_CONTROL = "ws://quincy.chuie.io:9999/api/ws/control/0"

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

local function wrapPeriph(side)
  local call = dp.call
  return function(method, ...)
    return call(side, method, ...)
  end
end

local monitors = { wrapPeriph("left") }

-- local monitors = {
--   wrapRemote("monitor_576", "bottom"),
--   wrapRemote("monitor_577", "bottom"),
--   wrapRemote("monitor_578", "bottom"),
--   wrapRemote("monitor_579", "bottom")
-- }

local joy = {
  ["b"] = 0,
  ["y"] = 1,
  ["select"] = 2,
  ["start"] = 3,
  ["up"] = 4,
  ["down"] = 5,
  ["left"] = 6,
  ["right"] = 7,
  ["a"] = 8,
  ["x"] = 9,
  ["l"] = 10,
  ["r"] = 11,
  ["l2"] = 12,
  ["r2"] = 13,
  ["l3"] = 14,
  ["r3"] = 15
}

local keyBindings = {
  [keys.x] = joy.a,
  [keys.z] = joy.b,
  [keys.a] = joy.y,
  [keys.s] = joy.x,
  [keys.q] = joy.l,
  [keys.w] = joy.r,
  [keys.up] = joy.up,
  [keys.down] = joy.down,
  [keys.left] = joy.left,
  [keys.right] = joy.right,
  [keys.enter] = joy.start,
  [keys.rightShift] = joy.select
}

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

local function run()
  clearMonitors()
  print("Connecting")

  local videoWS, err = http.websocket(JUROKU_VIDEO)
  if not videoWS then
    error(err)
  end

  local controlWS, err = http.websocket(JUROKU_CONTROL)
  if not controlWS then
    error(err)
  end

  local decoder = Decoder.new(monitors)

  for k, v in pairs(decoder) do
    print(k)
  end

  while true do
    local e = {os.pullEvent()}
    local event = e[1]

    if event == "websocket_failure" then
      error("Connection failed")
    elseif event == "websocket_message" then
      decoder:renderNextMonitor(0, e[3])
    elseif event == "key" and not e[3] and keyBindings[e[2]] ~= nil then
      -- key down and not being held
        controlWS.send(keyBindings[e[2]] .. " 1")
    elseif event == "key_up" and keyBindings[e[2]] ~= nil then
      controlWS.send(keyBindings[e[2]] .. " 0")
    end
  end

  if ws then ws.close() end
end

run()

clearMonitors()

print("Stream ended or something")