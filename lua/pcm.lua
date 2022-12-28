
local JUROKU_HOST = ""

if not debug then
    error("Missing debug API")
end
local dp
for i = 1, 16 do
    local name, value = debug.getupvalue(peripheral.getNames, i)
    if name == "native" then
        dp = value
        break
    end
end
if not dp then
    error("failed to get direct peripheral access")
end

-- local function wrapRemote(id, side)
--     local call = dp.call
--     return function(method, ...)
--         return call(side, "callRemote", id, method, ...)
--     end
-- end

-- local tapeA = wrapRemote("tape_drive_55", "back")

speaker = peripheral.wrap("top")

local function playAudio(data)
    if #data == 0 then
        return true
    end
    local tabular = {}
    for i = 1, #data do
    local val = data:byte(i)
    if val > 127 then
        val = val - 256
    end
    tabular[i] = val
    end

    return speaker.playAudio(tabular)
end

local function run()
    print("Connecting")

    local ok, endpoint = http.websocketAsync(JUROKU_HOST)
    if not ok then
        error("couldnt connect")
    end
    local ws
    local buffer = ""

    local first = false
    local timerID = nil
    local isEmpty = true

    while true do
        local e = {os.pullEvent()}
        local event = e[1]

        if event == "websocket_success" then
            ws = e[3]
            print("Connected!")
            ws.send("{\"id\": \"audio\", \"subscription\": 2}")
        elseif event == "websocket_failure" then
            error("Connection failed: " .. e[3])
        elseif event == "websocket_message" then
            -- print("rendering! " .. os.epoch("utc") .. " " .. type(e[3]))
            if e[3]:byte(1) == 2 then
                if not first then
                    print("first audio packet at: " .. os.epoch("utc"))
                    first = true
                end

                buffer = buffer .. e[3]:sub(3)

                if isEmpty then
                    if not playAudio(buffer) then
                        isEmpty = false
                    else
                        buffer = ""
                    end
                end

                -- print("audio packet length: " .. #(e[3]))
                -- print(e[3]:byte(2))
            end
        elseif event == "speaker_audio_empty" then
            if not playAudio(buffer) then
                isEmpty = false
            else
                buffer = ""
                isEmpty = true
            end
        end
    end

    if ws then
        ws.close()
    end
end

run()

print("Stream ended or something")
