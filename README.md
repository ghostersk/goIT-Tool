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
