local Decoder = require("jurokunext")

local JUROKU_HOST = ""

local name, dp = debug.getupvalue(peripheral.getNames, 2)
if name ~= "native" then
  error("failed to get direct peripheral access")
end

local function wrapRemote(id, side)
  local call = dp.call
  return function(method, ...)
    return call(side, "callRemote", id, method, ...)
  end
end

local function wrapLocal(side)
  local call = dp.call
  return function(method, ...)
    return call(side, method, ...)
  end
end

local monitors = {
  wrapLocal("right"),
--   wrapRemote("monitor_577", "bottom"),
--   wrapRemote("monitor_578", "bottom"),
--   wrapRemote("monitor_579", "bottom")
}

local monitorsNormal = {
  peripheral.wrap("monitor_576"),
  peripheral.wrap("monitor_577"),
  peripheral.wrap("monitor_578"),
  peripheral.wrap("monitor_579")
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

  local ok, endpoint = http.websocketAsync(JUROKU_HOST)
  if not ok then error("couldnt connect") end
  local ws
  local decoder = Decoder.new(monitors)

  for k, v in pairs(decoder) do
    print(k)
  end

  local count = 0

  parallel.waitForAll(function()
    while true do
      local e = {os.pullEvent()}
      local event = e[1]

      if event == "websocket_success" then
        ws = e[3]
        print("Connected!")
        ws.send("{\"id\": \"test\", \"subscription\": 1}")
      elseif event == "websocket_failure" then
        error("Connection failed")
      elseif event == "websocket_message" then
        --print("rendering! " .. os.epoch("utc") .. " " .. type(e[3]))
        if e[3]:byte(1) == 1 then
          if count == 0 then
            print("first packet at " .. os.epoch("utc"))
          end
          if count == 15 then
            print("15th frame at " .. os.epoch("utc"))
          end
          count = count+1
          decoder:renderNextMonitor(e[3]:byte(2), e[3]:sub(3, #e[3]))
        end
      end
    end
  end)

  if ws then ws.close() end
end

run()

clearMonitors()

print("Stream ended or something")
