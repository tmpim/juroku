-- local Decoder = require("jurokunext")

local JUROKU_HOST = ""

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

local tapeA = wrapRemote("tape_drive_55", "back")
local tapeB = wrapRemote("tape_drive_54", "back")

tapeA("setSpeed", 1.0)
tapeB("setSpeed", 1.0)

tapeA("seek", -tapeA("getPosition"))
tapeA("write", ("\0"):rep(tapeA("getSize")))
tapeA("seek", -tapeA("getPosition"))

tapeB("seek", -tapeB("getPosition"))
tapeB("write", ("\0"):rep(tapeB("getSize")))
tapeB("seek", -tapeB("getPosition"))

local tape = tapeA
local otherTape = tapeB

local playing = false

local buffer = ""

local function playAudio(data)
        print("playing:", #data)
        tape("seek", -tape("getPosition"))
        tape("write", data)
        tape("seek", -tape("getPosition"))
        tape("play")

        if tape == tapeA then
                tape = tapeB
                otherTape = tapeA
        else
                tape = tapeA
                otherTape = tapeB
        end

        return os.startTimer(2)
end

local function run()
        print("Connecting")

        local ok, endpoint = http.websocketAsync(JUROKU_HOST)
        if not ok then error("couldnt connect") end
        local ws
        local buffer = ""

        local first = false
        local timerID = nil

        while true do
                local e = {os.pullEvent()}
                local event = e[1]

                if event == "websocket_success" then
                        ws = e[3]
                        print("Connected!")
                        ws.send("{\"id\": \"audio\", \"subscription\": 2}")
                elseif event == "websocket_failure" then
                        error("Connection failed")
                elseif event == "websocket_message" then
                        --print("rendering! " .. os.epoch("utc") .. " " .. type(e[3]))
                        if e[3]:byte(1) == 2 then
                                if not first then
                                        print("first audio packet at: " .. os.epoch("utc"))
                                        first = true
                                end

                                print("audio packet length: " .. #(e[3]))
                                print(e[3]:byte(2))

                                buffer = buffer .. e[3]:sub(3)

                                if e[3]:byte(2) == 1 then
                                        timerID = playAudio(buffer)
                                        buffer = ""
                                end

                                -- timerID = playAudio(e[3]:sub(2))
                        end
                elseif event == "timer" and e[2] == timerID then
                        tape("stop")
                end
        end

        if ws then ws.close() end
end

run()

clearMonitors()

print("Stream ended or something")
