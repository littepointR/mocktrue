#!/usr/bin/osascript
-- 截取最前面的 PortWeave 窗口
on run argv
  set outputPath to item 1 of argv
  do shell script "screencapture -l$(osascript -e 'tell app \"System Events\" to id of window 1 of process \"portweave\"') " & quoted form of outputPath
end run
