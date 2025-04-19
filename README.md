# goIT-Tool
IT Multitool

```powershell
go mod init win-multitool
go get github.com/getlantern/systray
go get github.com/kardianos/service
go get github.com/lxn/walk

# icon generator -16x16 from png:
# https://convertico.com/
go mod tidy
# install rsrc - will be in default `~\go\bin`  you may need to add it to path
go install github.com/akavel/rsrc
rsrc.exe -manifest main.manifest -ico repair.ico -o rsrc.syso
# Without the -ldflags... it will fail to open the GUI
go build -ldflags="-H windowsgui"

# add as service - not tested yet with this app.
sc create HostsEditorService binPath="C:\temp\projects\go\win-multitool\win-multitool.exe" start= auto
```
## Current issues:
1. The network interface needs overhaul - it should have network interfaces names in tabs, but it has the actual values...
![image](https://github.com/user-attachments/assets/3326f3a2-eb64-4fdc-a30d-84ce4be8ddd6)
2. Verify this actually changes the settings properly
3. Fix the GUI to work all the time, not just first time it is opened!


## Currently working:
1. When you start it as administrator, user can open the Hosts file and edit it using Notepad.
![image](https://github.com/user-attachments/assets/dd354287-dc86-42c7-8b99-00898f0e5b63)
![image](https://github.com/user-attachments/assets/8b960c04-ea07-4003-84b1-51ad8af275a5)
