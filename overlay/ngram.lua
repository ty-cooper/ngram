-- Ngram Capture Overlay for Hammerspoon
-- Install: symlink or copy to ~/.hammerspoon/ngram.lua
-- In init.lua: require("ngram")

local M = {}

-- Configuration (override in init.lua if needed)
M.vaultPath = os.getenv("NGRAM_VAULT_PATH") or os.getenv("HOME") .. "/.obsidian.ngram"
M.nBinary = "n"

-- State
local captureSession = nil
local overlay = nil
local textBuffer = ""

----------------------------------------------------------------------
-- Mode 1: Mixed-Media Capture Session (Cmd+Shift+N)
----------------------------------------------------------------------

local function startMixedSession()
    if captureSession then return end

    local ts = os.time()
    local sessionDir = M.vaultPath .. "/_inbox/" .. ts .. "-capture-session"
    os.execute("mkdir -p " .. sessionDir)

    captureSession = {
        dir = sessionDir,
        items = {},
        screenshotCount = 0,
        startTime = os.date("!%Y-%m-%dT%H:%M:%SZ"),
    }

    -- Read .boxrc context
    local boxrc = readBoxRC()

    showOverlay("Ngram Capture", boxrc)
end

local function captureScreenshot()
    if not captureSession then return end

    -- Hide overlay during capture
    if overlay then overlay:hide() end
    hs.timer.doAfter(0.2, function()
        captureSession.screenshotCount = captureSession.screenshotCount + 1
        local filename = string.format("ss-%03d.png", captureSession.screenshotCount)
        local filepath = captureSession.dir .. "/" .. filename

        -- Use screencapture for region select
        hs.task.new("/usr/sbin/screencapture", function(exitCode)
            if exitCode == 0 then
                table.insert(captureSession.items, {
                    type = "screenshot",
                    file = filename,
                    timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ"),
                })
                updateOverlayContent()
            end
            if overlay then overlay:show() end
        end, {"-i", filepath}):start()
    end)
end

local function addTextBlock()
    if not captureSession or textBuffer == "" then return end

    table.insert(captureSession.items, {
        type = "text",
        content = textBuffer,
        timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ"),
    })
    textBuffer = ""
    updateOverlayContent()
end

local function finishSession()
    if not captureSession then return end

    -- Write manifest.yml
    local manifest = "session_id: \"" .. captureSession.startTime .. "\"\n"
    manifest = manifest .. "capture_mode: \"mixed\"\n"
    manifest = manifest .. "items:\n"

    for _, item in ipairs(captureSession.items) do
        if item.type == "screenshot" then
            manifest = manifest .. "  - type: screenshot\n"
            manifest = manifest .. "    file: " .. item.file .. "\n"
            manifest = manifest .. "    timestamp: \"" .. item.timestamp .. "\"\n"
        elseif item.type == "text" then
            manifest = manifest .. "  - type: text\n"
            manifest = manifest .. "    content: \"" .. item.content:gsub('"', '\\"') .. "\"\n"
            manifest = manifest .. "    timestamp: \"" .. item.timestamp .. "\"\n"
        end
    end

    local f = io.open(captureSession.dir .. "/manifest.yml", "w")
    if f then
        f:write(manifest)
        f:close()
    end

    local count = #captureSession.items
    captureSession = nil
    hideOverlay()

    hs.notify.new({title = "Ngram", informativeText = count .. " items captured"}):send()
end

local function abortSession()
    if captureSession then
        os.execute("rm -rf " .. captureSession.dir)
        captureSession = nil
    end
    hideOverlay()
end

----------------------------------------------------------------------
-- Mode 2: Text-Only Note (Cmd+Shift+M)
----------------------------------------------------------------------

local function startTextNote()
    local boxrc = readBoxRC()
    showTextOverlay("Ngram Note", boxrc)
end

local function saveTextNote(text)
    if text == "" then return end

    local ts = os.time()
    local slug = text:sub(1, 50):lower():gsub("[^a-z0-9]+", "-"):gsub("^-+", ""):gsub("-+$", "")
    local filename = ts .. "-" .. slug .. ".md"
    local filepath = M.vaultPath .. "/_inbox/" .. filename

    local frontmatter = "---\n"
    frontmatter = frontmatter .. 'captured: "' .. os.date("!%Y-%m-%dT%H:%M:%SZ") .. '"\n'
    frontmatter = frontmatter .. 'source: "overlay"\n'
    frontmatter = frontmatter .. 'capture_mode: "text"\n'
    frontmatter = frontmatter .. "---\n\n"

    local f = io.open(filepath, "w")
    if f then
        f:write(frontmatter .. text .. "\n")
        f:close()
    end

    hideOverlay()
    hs.notify.new({title = "Ngram", informativeText = "Note captured"}):send()
end

----------------------------------------------------------------------
-- Mode 3: Quick Screenshot (Cmd+Shift+S)
----------------------------------------------------------------------

local function quickScreenshot()
    local ts = os.time()
    local filename = ts .. "-screenshot.png"
    local filepath = M.vaultPath .. "/_inbox/" .. filename

    hs.task.new("/usr/sbin/screencapture", function(exitCode)
        if exitCode == 0 then
            hs.notify.new({title = "Ngram", informativeText = "Screenshot captured"}):send()
        end
    end, {"-i", filepath}):start()
end

----------------------------------------------------------------------
-- Overlay UI (Hammerspoon webview)
----------------------------------------------------------------------

function showOverlay(title, boxrc)
    if overlay then overlay:delete() end

    local screen = hs.screen.mainScreen():frame()
    local w, h = 600, 500
    local x = screen.x + screen.w - w - 20
    local y = screen.y + 40

    overlay = hs.webview.new({x = x, y = y, w = w, h = h})
    overlay:windowStyle({"titled", "closable", "utility"})
    overlay:level(hs.drawing.windowLevels.floating)
    overlay:title(title)
    overlay:html(buildOverlayHTML(boxrc))
    overlay:show()
end

function showTextOverlay(title, boxrc)
    if overlay then overlay:delete() end

    local screen = hs.screen.mainScreen():frame()
    local w, h = 600, 400
    local x = screen.x + screen.w - w - 20
    local y = screen.y + 40

    overlay = hs.webview.new({x = x, y = y, w = w, h = h})
    overlay:windowStyle({"titled", "closable", "utility"})
    overlay:level(hs.drawing.windowLevels.floating)
    overlay:title(title)
    overlay:html(buildTextOverlayHTML(boxrc))
    overlay:show()
end

function hideOverlay()
    if overlay then
        overlay:delete()
        overlay = nil
    end
    textBuffer = ""
end

function updateOverlayContent()
    if not overlay or not captureSession then return end
    overlay:html(buildOverlayHTML(nil))
end

function buildOverlayHTML(boxrc)
    local items_html = ""
    if captureSession then
        for _, item in ipairs(captureSession.items) do
            if item.type == "screenshot" then
                items_html = items_html .. '<div class="item ss">[Screenshot: ' .. item.file .. ']</div>'
            elseif item.type == "text" then
                items_html = items_html .. '<div class="item text">' .. item.content .. '</div>'
            end
        end
    end

    return [[
    <html><head><style>
    body { font-family: -apple-system; background: #1a1a2e; color: #eee; padding: 16px; margin: 0; }
    .item { padding: 8px; margin: 4px 0; border-radius: 4px; }
    .ss { background: #16213e; }
    .text { background: #0f3460; font-style: italic; }
    .footer { position: fixed; bottom: 0; left: 0; right: 0; padding: 12px 16px; background: #0a0a1a; color: #888; font-size: 12px; }
    </style></head><body>
    ]] .. items_html .. [[
    <div class="footer">[Cmd+S] screenshot  [Enter] add text  [Cmd+Enter] finish  [Esc] abort</div>
    </body></html>
    ]]
end

function buildTextOverlayHTML(boxrc)
    return [[
    <html><head><style>
    body { font-family: -apple-system; background: #1a1a2e; color: #eee; padding: 16px; margin: 0; }
    textarea { width: 100%; height: 280px; background: #16213e; color: #eee; border: 1px solid #333; padding: 12px; font-size: 14px; resize: none; font-family: -apple-system; }
    .footer { position: fixed; bottom: 0; left: 0; right: 0; padding: 12px 16px; background: #0a0a1a; color: #888; font-size: 12px; }
    </style></head><body>
    <textarea id="note" autofocus placeholder="Type your note..."></textarea>
    <div class="footer">[Cmd+Enter] save  [Cmd+S] screenshot  [Esc] abort</div>
    </body></html>
    ]]
end

----------------------------------------------------------------------
-- Helpers
----------------------------------------------------------------------

function readBoxRC()
    local cwd = hs.execute("pwd"):gsub("\n", "")
    local f = io.open(cwd .. "/.boxrc", "r")
    if not f then return nil end
    local content = f:read("*a")
    f:close()
    return content
end

----------------------------------------------------------------------
-- Hotkey Bindings
----------------------------------------------------------------------

-- Unified capture panel (Cmd+Shift+N) — shows mode picker.
-- Direct shortcuts preserved for power users.
local function showCapturePicker()
    local choices = {
        {text = "[N] Mixed-media session (screenshots + text)", subText = "Cmd+Shift+N again to skip picker"},
        {text = "[M] Text-only note", subText = "Quick brain dump"},
        {text = "[S] Quick screenshot", subText = "Region select, fire and forget"},
    }
    local chooser = hs.chooser.new(function(choice)
        if not choice then return end
        local idx = choice.idx
        if idx == 1 then startMixedSession()
        elseif idx == 2 then startTextNote()
        elseif idx == 3 then quickScreenshot()
        end
    end)
    for i, c in ipairs(choices) do
        choices[i] = {text = c.text, subText = c.subText, idx = i}
    end
    chooser:choices(choices)
    chooser:show()
end

hs.hotkey.bind({"cmd", "shift"}, "N", function()
    if captureSession then
        -- Already in a session, don't open picker.
        return
    end
    showCapturePicker()
end)
hs.hotkey.bind({"cmd", "shift"}, "M", startTextNote)
hs.hotkey.bind({"cmd", "shift"}, "S", quickScreenshot)

-- Session keybindings (active during mixed-media session)
hs.hotkey.bind({"cmd"}, "S", function()
    if captureSession then captureScreenshot() end
end)
hs.hotkey.bind({"cmd"}, "return", function()
    if captureSession then finishSession()
    elseif overlay then saveTextNote(textBuffer) end
end)
hs.hotkey.bind({}, "escape", function()
    if captureSession then abortSession()
    elseif overlay then hideOverlay() end
end)

----------------------------------------------------------------------
-- Module
----------------------------------------------------------------------

return M
