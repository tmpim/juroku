local JUROKU_CONTROL = ".../api/ws/control/0"

local name, dp = debug.getupvalue(peripheral.getNames, 2)
if name ~= "native" then
  error("failed to get direct peripheral access")
end

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

local function run()
  print("Connecting")

  local controlWS, err = http.websocket(JUROKU_CONTROL)
  if not controlWS then
    error(err)
  end

  local isEmpty = true
  local buffer = ""

  while true do
    local e = {os.pullEvent()}
    local event, url = e[1], e[2]

    if event == "websocket_failure" then
      print("Connection failed, rebooting in 3 seconds...")
      sleep(3)
      os.reboot()
    elseif event == "websocket_closed" then
      os.reboot()
    elseif event == "key" and not e[3] and keyBindings[e[2]] ~= nil then
      -- key down and not being held
        controlWS.send(keyBindings[e[2]] .. " 1")
    elseif event == "speaker_audio_empty" then
      if not playAudio(buffer) then
        isEmpty = false
      else
        buffer = ""
        isEmpty = true
      end
    elseif event == "key_up" and keyBindings[e[2]] ~= nil then
      controlWS.send(keyBindings[e[2]] .. " 0")
    end
  end

  if ws then ws.close() end
end

run()

print("Stream ended or something")
