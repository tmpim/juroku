local hex = {"0","1","2","3","4","5","6","7","8","9","a","b","c","d","e","f"}

function draw(t, file)
	local f = fs.open(file, "rb")

	while true do
		local start = os.clock()

		local first = f.read()
		if first == nil then
			return
		end

		local width = first * 0x100 + f.read()
		local height = f.read() * 0x100 + f.read()
		for i = 1, 16 do
			t.setPaletteColor(2^(i-1), f.read() * 0x10000 + f.read() * 0x100 + f.read())
		end

		local x, y = t.getCursorPos()

		for row = 1, height do
			local fg = ""
			local bg = ""
			local txt = ""
			for col = 1, width do
				local color = f.read()
				fg = fg .. hex[math.floor(color / 0x10) + 1]
				bg = bg .. hex[bit.band(color, 0xF) + 1]
				txt = txt .. string.char(f.read())
			end
			t.setCursorPos(x, y + row - 1)
			t.blit(txt, fg, bg)
		end

		sleep((start + 0.1) - os.clock())
	end
end
