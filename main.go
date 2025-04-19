package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/getlantern/systray"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type NetworkInterface struct {
	Name          string
	IsDHCP        bool
	IP            string
	SubnetMask    string
	Gateway       string
	DNS           string
	DNSSuffix     string
	RegisterInDNS bool
}

func runSystemTray() {
	// Set the icon for the system tray
	iconData := getIcon()
	if iconData == nil {
		log.Println("Using default icon due to icon loading failure")
	} else {
		systray.SetIcon(iconData)
	}
	systray.SetTitle("Hosts Editor")
	systray.SetTooltip("Hosts File Editor Service")

	// Menu items
	mEditHosts := systray.AddMenuItem("Edit Hosts", "Edit the hosts file")
	systray.AddSeparator()
	mManageNetwork := systray.AddMenuItem("Manage Network", "Manage network interfaces")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mEditHosts.ClickedCh:
				if err := openHostsFile(); err != nil {
					log.Printf("Failed to open hosts file: %v", err)
				}
			case <-mManageNetwork.ClickedCh:
				go func() {
					if err := showNetworkGUI(); err != nil {
						log.Printf("Failed to show network GUI: %v", err)
					}
				}()
			case <-mQuit.ClickedCh:
				if confirmQuit() {
					systray.Quit()
					os.Exit(0)
				}
			}
		}
	}()
}

func getIcon() []byte {
	// Load repair.ico from the project directory
	iconPath := filepath.Join("repair.ico")
	data, err := os.ReadFile(iconPath)
	if err != nil {
		log.Printf("Failed to load icon: %v", err)
		return nil
	}
	if len(data) == 0 {
		log.Println("Icon file is empty")
		return nil
	}
	// Basic .ico file validation (check for ICO header)
	if len(data) < 6 || data[0] != 0x00 || data[1] != 0x00 || data[2] != 0x01 || data[3] != 0x00 {
		log.Println("Invalid .ico file format")
		return nil
	}
	log.Printf("Loaded icon file, size: %d bytes, first 6 bytes: %x", len(data), data[:6])
	return data
}

func openHostsFile() error {
	hostsPath := `C:\Windows\System32\drivers\etc\hosts`
	cmd := exec.Command("notepad.exe", hostsPath)
	return cmd.Start()
}

func confirmQuit() bool {
	const MB_YESNO = 0x00000004
	const MB_ICONQUESTION = 0x00000020
	const IDYES = 6

	hwnd, err := windows.MessageBox(0, windows.StringToUTF16Ptr("Are you sure you want to close this application?"),
		windows.StringToUTF16Ptr("Confirm Exit"), MB_YESNO|MB_ICONQUESTION)
	if err != nil {
		log.Printf("MessageBox failed: %v", err)
		return false
	}
	return hwnd == IDYES
}

func showNetworkGUI() error {
	interfaces, err := getNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %v", err)
	}

	var mw *walk.MainWindow
	var tabs *walk.TabWidget

	// Use a slice of pointers to interface states for dynamic updates
	interfaceStates := make([]*NetworkInterface, 0, len(interfaces))
	for i := range interfaces {
		interfaceStates = append(interfaceStates, &interfaces[i])
	}

	var tabPages []TabPage
	for _, iface := range interfaceStates {
		if strings.ToLower(iface.Name) == "loopback" {
			continue
		}
		page, err := createInterfaceTab(iface)
		if err != nil {
			log.Printf("Failed to create tab for %s: %v", iface.Name, err)
			continue
		}
		tabPages = append(tabPages, page)
	}

	if len(tabPages) == 0 {
		return fmt.Errorf("no valid network interfaces found")
	}

	err = MainWindow{
		AssignTo: &mw,
		Title:    "Network Interface Manager",
		MinSize:  Size{Width: 400, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			TabWidget{
				AssignTo: &tabs,
				Pages:    tabPages,
			},
		},
	}.Create()
	if err != nil {
		return fmt.Errorf("failed to create main window: %v", err)
	}

	mw.Run()
	return nil
}

func createInterfaceTab(iface *NetworkInterface) (TabPage, error) {
	var dhcpCheckBox *walk.CheckBox
	var ipEdit, maskEdit, gatewayEdit, dnsEdit, suffixEdit *walk.LineEdit
	var registerDNSCheckBox *walk.CheckBox
	var warningLabel *walk.Label
	var saveButton, cancelButton *walk.PushButton

	updateFields := func() {
		readonly := dhcpCheckBox.Checked()
		ipEdit.SetReadOnly(readonly)
		maskEdit.SetReadOnly(readonly)
		gatewayEdit.SetReadOnly(readonly)
		dnsEdit.SetReadOnly(readonly)
		// DNS Suffix and RegisterInDNS are always editable
		warningLabel.SetVisible(strings.HasPrefix(ipEdit.Text(), "169.254.") && dhcpCheckBox.Checked())
	}

	return TabPage{
		Title:  iface.Name,
		Layout: Grid{Columns: 2},
		Children: []Widget{
			Label{Text: "DHCP Enabled:"},
			CheckBox{
				AssignTo: &dhcpCheckBox,
				Checked:  iface.IsDHCP,
				OnCheckedChanged: func() {
					updateFields()
				},
			},
			Label{Text: "IP Address:"},
			LineEdit{
				AssignTo: &ipEdit,
				Text:     iface.IP,
				ReadOnly: iface.IsDHCP,
			},
			Label{Text: "Subnet Mask:"},
			LineEdit{
				AssignTo: &maskEdit,
				Text:     iface.SubnetMask,
				ReadOnly: iface.IsDHCP,
			},
			Label{Text: "Gateway:"},
			LineEdit{
				AssignTo: &gatewayEdit,
				Text:     iface.Gateway,
				ReadOnly: iface.IsDHCP,
			},
			Label{Text: "DNS Server:"},
			LineEdit{
				AssignTo: &dnsEdit,
				Text:     iface.DNS,
				ReadOnly: iface.IsDHCP,
			},
			Label{Text: "DNS Suffix:"},
			LineEdit{
				AssignTo: &suffixEdit,
				Text:     iface.DNSSuffix,
			},
			Label{Text: "Register in DNS:"},
			CheckBox{
				AssignTo: &registerDNSCheckBox,
				Checked:  iface.RegisterInDNS,
			},
			Label{
				AssignTo:   &warningLabel,
				Text:       "Warning: IP address indicates no DHCP server connection (169.254.x.x)",
				Visible:    strings.HasPrefix(iface.IP, "169.254.") && iface.IsDHCP,
				ColumnSpan: 2,
			},
			PushButton{
				AssignTo: &saveButton,
				Text:     "Save",
				OnClicked: func() {
					err := saveInterfaceSettings(*iface, dhcpCheckBox.Checked(), ipEdit.Text(), maskEdit.Text(), gatewayEdit.Text(), dnsEdit.Text(), suffixEdit.Text(), registerDNSCheckBox.Checked())
					if err != nil {
						walk.MsgBox(nil, "Error", fmt.Sprintf("Failed to save settings: %v", err), walk.MsgBoxIconError)
					} else {
						walk.MsgBox(nil, "Success", "Settings saved successfully", walk.MsgBoxIconInformation)
						// Update the interface state
						iface.IsDHCP = dhcpCheckBox.Checked()
						iface.IP = ipEdit.Text()
						iface.SubnetMask = maskEdit.Text()
						iface.Gateway = gatewayEdit.Text()
						iface.DNS = dnsEdit.Text()
						iface.DNSSuffix = suffixEdit.Text()
						iface.RegisterInDNS = registerDNSCheckBox.Checked()
						updateFields()
					}
				},
			},
			PushButton{
				AssignTo: &cancelButton,
				Text:     "Cancel",
				OnClicked: func() {
					ipEdit.SetText(iface.IP)
					maskEdit.SetText(iface.SubnetMask)
					gatewayEdit.SetText(iface.Gateway)
					dnsEdit.SetText(iface.DNS)
					suffixEdit.SetText(iface.DNSSuffix)
					dhcpCheckBox.SetChecked(iface.IsDHCP)
					registerDNSCheckBox.SetChecked(iface.RegisterInDNS)
					updateFields()
				},
			},
		},
	}, nil
}

func getNetworkInterfaces() ([]NetworkInterface, error) {
	var interfaces []NetworkInterface

	// Get interface configurations
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "config")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentIface *NetworkInterface
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
			if currentIface != nil {
				interfaces = append(interfaces, *currentIface)
			}
			name := strings.TrimSuffix(line, ":")
			currentIface = &NetworkInterface{Name: name}
		} else if currentIface != nil && strings.HasPrefix(line, "   ") {
			if strings.Contains(line, "DHCP enabled:") {
				currentIface.IsDHCP = strings.Contains(line, "Yes")
			} else if strings.Contains(line, "IP Address:") {
				currentIface.IP = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Subnet Mask:") {
				currentIface.SubnetMask = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Default Gateway:") {
				currentIface.Gateway = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.Contains(line, "Statically Configured DNS Servers:") || strings.Contains(line, "DNS servers configured through DHCP:") {
				currentIface.DNS = strings.TrimSpace(strings.Split(line, ":")[1])
				if currentIface.DNS == "None" {
					currentIface.DNS = ""
				}
			}
		}
	}
	if currentIface != nil {
		interfaces = append(interfaces, *currentIface)
	}

	// Get DNS suffix and registration from registry
	for i, iface := range interfaces {
		ifaceGUID, err := getInterfaceGUID(iface.Name)
		if err == nil {
			keyPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\` + ifaceGUID
			k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
			if err == nil {
				interfaces[i].DNSSuffix, _, _ = k.GetStringValue("Domain")
				register, _, err := k.GetIntegerValue("RegisterAdapterName")
				if err == nil && register == 1 {
					interfaces[i].RegisterInDNS = true
				}
				k.Close()
			}
		}
	}

	return interfaces, nil
}

func getInterfaceGUID(ifaceName string) (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return "", fmt.Errorf("failed to open registry: %v", err)
	}
	defer k.Close()

	guids, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return "", fmt.Errorf("failed to read subkeys: %v", err)
	}

	for _, guid := range guids {
		subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\`+guid, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		name, _, err := subKey.GetStringValue("Name")
		if err == nil && name == ifaceName {
			subKey.Close()
			return guid, nil
		}
		subKey.Close()
	}

	return "", fmt.Errorf("interface GUID not found for %s", ifaceName)
}

func saveInterfaceSettings(iface NetworkInterface, isDHCP bool, ip, mask, gateway, dns, suffix string, registerInDNS bool) error {
	if isDHCP {
		cmd := exec.Command("netsh", "interface", "ipv4", "set", "address", "name="+iface.Name, "source=dhcp")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable DHCP for IP: %v", err)
		}
		cmd = exec.Command("netsh", "interface", "ipv4", "set", "dnsservers", "name="+iface.Name, "source=dhcp")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable DHCP for DNS: %v", err)
		}
	} else {
		if !isValidIPv4(ip) || !isValidIPv4(mask) {
			return fmt.Errorf("invalid IP or subnet mask")
		}
		args := []string{"interface", "ipv4", "set", "address", "name=" + iface.Name, "source=static", "address=" + ip, "mask=" + mask}
		if gateway != "" && isValidIPv4(gateway) {
			args = append(args, "gateway="+gateway)
		}
		cmd := exec.Command("netsh", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set static IP: %v", err)
		}
		if dns != "" && isValidIPv4(dns) {
			cmd = exec.Command("netsh", "interface", "ipv4", "set", "dnsservers", "name="+iface.Name, "source=static", "address="+dns, "validate=no")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set DNS: %v", err)
			}
		}
	}

	// Set DNS suffix and registration
	if suffix != iface.DNSSuffix || registerInDNS != iface.RegisterInDNS {
		guid, err := getInterfaceGUID(iface.Name)
		if err != nil {
			return fmt.Errorf("failed to get interface GUID: %v", err)
		}
		keyPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\` + guid
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("failed to open registry key: %v", err)
		}
		defer k.Close()

		if err := k.SetStringValue("Domain", suffix); err != nil {
			return fmt.Errorf("failed to set DNS suffix: %v", err)
		}
		registerValue := uint32(0)
		if registerInDNS {
			registerValue = 1
		}
		if err := k.SetDWordValue("RegisterAdapterName", registerValue); err != nil {
			return fmt.Errorf("failed to set DNS registration: %v", err)
		}
	}

	return nil
}

func isValidIPv4(ip string) bool {
	re := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !re.MatchString(ip) {
		return false
	}
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}
	return true
}

func main() {
	systray.Run(runSystemTray, func() {})
}
