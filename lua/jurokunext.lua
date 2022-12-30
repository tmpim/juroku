local Decoder = {}

function Decoder.new(monitors)
  local self = {monitors = monitors}
  setmetatable(self, {__index = Decoder})
  return self
end

function Decoder:renderNextMonitor(monNum, frame)
  local pos = 1
  local wa, wb, ha, hb = frame:byte(pos, pos+3)
  pos = pos + 4

  local width = wa * 0x100 + wb
  local height = ha * 0x100 + hb

  local t = self.monitors[monNum+1]

  for i = 1, 16 do
    local r, g, b = frame:byte(pos, pos+2)
    t("setPaletteColor", 2^(i-1), r * 0x10000 + g * 0x100 + b)
    pos = pos + 3
  end

  for row = 1, height do
    t("setCursorPos", 1, row)
    t("blit", frame:sub(pos, pos + width - 1), frame:sub(pos + width, pos + width*2 - 1), frame:sub(pos + width*2, pos + width*3 - 1))
    pos = pos + width * 3
  end
end

return Decoder
