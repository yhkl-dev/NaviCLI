package device

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type AudioDeviceType int

const (
	AudioDeviceUnknown    AudioDeviceType = iota
	AudioDeviceBuiltIn                    // Built-in speakers
	AudioDeviceBluetooth                  // Bluetooth audio device
	AudioDeviceUSB                        // USB audio device
	AudioDeviceHDMI                       // HDMI audio
	AudioDeviceHeadphones                 // Wired headphones
)

type AudioDeviceInfo struct {
	Name       string
	DeviceType AudioDeviceType
	Transport  string
}

type AudioMonitor struct {
	ctx               context.Context
	checkInterval     time.Duration
	onDisconnect      func()
	lastDevice        *AudioDeviceInfo
	wasExternalDevice bool
	supported         bool
}

func NewAudioMonitor(ctx context.Context, onDisconnect func()) *AudioMonitor {
	return &AudioMonitor{
		ctx:           ctx,
		checkInterval: 500 * time.Millisecond,
		onDisconnect:  onDisconnect,
		supported:     runtime.GOOS == "darwin",
	}
}

func (m *AudioMonitor) Start() {
	if !m.supported {
		log.Println("Audio monitor disabled: unsupported platform")
		return
	}

	go m.monitorLoop()
}

func (m *AudioMonitor) monitorLoop() {
	if !m.supported {
		return
	}

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	// Get initial device state
	records := getAudioDeviceRecords()
	m.lastDevice = selectCurrentDevice(records)
	if m.lastDevice == nil {
		m.lastDevice = m.getAudioDeviceViaOsascript()
	}
	if m.lastDevice != nil {
		m.wasExternalDevice = isExternalDevice(m.lastDevice.DeviceType)
		log.Printf("Initial audio device: %s (type: %v)", m.lastDevice.Name, m.lastDevice.DeviceType)
	}

	for {
		select {
		case <-ticker.C:
			records := getAudioDeviceRecords()
			currentDevice := selectCurrentDevice(records)
			if currentDevice == nil {
				currentDevice = m.getAudioDeviceViaOsascript()
			}
			if currentDevice == nil {
				continue
			}

			// Check if device changed from external to built-in
			if m.wasExternalDevice && !isExternalDevice(currentDevice.DeviceType) {
				if m.onDisconnect != nil {
					m.onDisconnect()
				}
				m.lastDevice = currentDevice
				m.wasExternalDevice = false
				continue
			}

			// For current external device, verify it is still reported as connected
			if m.lastDevice != nil && isExternalDevice(m.lastDevice.DeviceType) {
				connectedDevices := getAllConnectedExternalDevices(records)
				found := false
				for device := range connectedDevices {
					if deviceNameMatches(m.lastDevice.Name, device) {
						found = true
						break
					}
				}

				if !found {
					newDevice := currentDevice
					if deviceNameMatches(newDevice.Name, m.lastDevice.Name) {
						newDevice = m.getAudioDeviceViaOsascript()
					}
					if newDevice != nil && !deviceNameMatches(newDevice.Name, m.lastDevice.Name) {
						if m.onDisconnect != nil {
							m.onDisconnect()
						}
						m.lastDevice = newDevice
						m.wasExternalDevice = isExternalDevice(newDevice.DeviceType)
						continue
					}
				}
			}

			m.lastDevice = currentDevice
			m.wasExternalDevice = isExternalDevice(currentDevice.DeviceType)

		case <-m.ctx.Done():
			log.Println("Audio monitor stopped")
			return
		}
	}
}

func (m *AudioMonitor) getCurrentAudioDevice() *AudioDeviceInfo {
	if !m.supported {
		return nil
	}

	records := getAudioDeviceRecords()
	if info := selectCurrentDevice(records); info != nil {
		return info
	}

	return m.getAudioDeviceViaOsascript()
}

func (m *AudioMonitor) getAudioDeviceViaOsascript() *AudioDeviceInfo {
	if !m.supported {
		return nil
	}

	script := `do shell script "SwitchAudioSource -c 2>/dev/null || echo 'unknown'"`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return m.getAudioDeviceViaSystemPrefs()
	}

	deviceName := strings.TrimSpace(string(output))
	if deviceName == "" || deviceName == "unknown" {
		return m.getAudioDeviceViaSystemPrefs()
	}

	return &AudioDeviceInfo{
		Name:       deviceName,
		DeviceType: detectDeviceType(deviceName, ""),
		Transport:  "",
	}
}

func (m *AudioMonitor) getAudioDeviceViaSystemPrefs() *AudioDeviceInfo {
	if !m.supported {
		return nil
	}

	cmd := exec.Command("sh", "-c",
		`ioreg -c IOAudioEngine -k IOAudioEngineState | grep -E '"IOAudioEngineDescription"' | head -1`)
	output, err := cmd.Output()
	if err != nil {
		return &AudioDeviceInfo{
			Name:       "Unknown",
			DeviceType: AudioDeviceUnknown,
			Transport:  "",
		}
	}

	deviceName := extractDeviceName(string(output))
	return &AudioDeviceInfo{
		Name:       deviceName,
		DeviceType: detectDeviceType(deviceName, ""),
		Transport:  "",
	}
}

func selectCurrentDevice(records []audioDeviceRecord) *AudioDeviceInfo {
	var fallback *AudioDeviceInfo

	for _, rec := range records {
		info := &AudioDeviceInfo{
			Name:       rec.Name,
			DeviceType: detectDeviceType(rec.Name, rec.Transport),
			Transport:  rec.Transport,
		}

		if rec.IsDefaultOutput && rec.IsConnected {
			return info
		}
		if rec.IsDefaultOutput && fallback == nil {
			fallback = info
			continue
		}
		if fallback == nil && rec.IsConnected {
			fallback = info
		}
	}

	return fallback
}

func extractDeviceName(output string) string {
	output = strings.TrimSpace(output)
	if idx := strings.Index(output, "="); idx != -1 {
		name := strings.TrimSpace(output[idx+1:])
		name = strings.Trim(name, `"`)
		return name
	}
	return "Unknown"
}

func detectDeviceType(name, transport string) AudioDeviceType {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	transportLower := strings.ToLower(strings.TrimSpace(transport))

	switch transportLower {
	case "bluetooth", "wireless", "ble":
		return AudioDeviceBluetooth
	case "usb", "usb audio", "usb audio device", "usbaudio":
		return AudioDeviceUSB
	case "hdmi", "displayport", "display port", "thunderbolt":
		return AudioDeviceHDMI
	case "built-in", "internal":
		return AudioDeviceBuiltIn
	case "headphone", "headset", "line out", "3.5mm", "analog":
		return AudioDeviceHeadphones
	}

	if strings.Contains(nameLower, "bluetooth") ||
		strings.Contains(nameLower, "airpods") ||
		strings.Contains(nameLower, "beats") ||
		strings.Contains(nameLower, "sony wh") ||
		strings.Contains(nameLower, "sony wf") ||
		strings.Contains(nameLower, "bose") ||
		strings.Contains(nameLower, "jabra") ||
		strings.Contains(nameLower, "sennheiser") ||
		strings.Contains(nameLower, "jbl") ||
		strings.Contains(nameLower, "marshall") ||
		strings.Contains(nameLower, "b&o") ||
		strings.Contains(nameLower, "bang & olufsen") {
		return AudioDeviceBluetooth
	}

	if strings.Contains(nameLower, "built-in") ||
		strings.Contains(nameLower, "internal") ||
		strings.Contains(nameLower, "macbook") ||
		strings.Contains(nameLower, "imac") ||
		strings.Contains(nameLower, "mac mini") ||
		strings.Contains(nameLower, "mac pro") ||
		strings.Contains(nameLower, "speakers") {
		return AudioDeviceBuiltIn
	}

	if strings.Contains(nameLower, "usb") ||
		strings.Contains(nameLower, "dac") ||
		strings.Contains(nameLower, "audio interface") {
		return AudioDeviceUSB
	}

	if strings.Contains(nameLower, "hdmi") ||
		strings.Contains(nameLower, "displayport") ||
		strings.Contains(nameLower, "display audio") {
		return AudioDeviceHDMI
	}

	if strings.Contains(nameLower, "headphone") ||
		strings.Contains(nameLower, "headset") ||
		strings.Contains(nameLower, "external headphones") {
		return AudioDeviceHeadphones
	}

	return AudioDeviceUnknown
}

func isExternalDevice(deviceType AudioDeviceType) bool {
	return deviceType == AudioDeviceBluetooth ||
		deviceType == AudioDeviceUSB ||
		deviceType == AudioDeviceHDMI ||
		deviceType == AudioDeviceHeadphones
}

type audioDeviceRecord struct {
	Name            string
	Transport       string
	IsDefaultOutput bool
	IsConnected     bool
}

func getAudioDeviceRecords() []audioDeviceRecord {
	if runtime.GOOS != "darwin" {
		return nil
	}

	cmd := exec.Command("system_profiler", "SPAudioDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to get audio devices: %v", err)
		return nil
	}

	records, err := parseAudioDeviceRecords(output)
	if err != nil {
		log.Printf("Failed to parse audio devices JSON: %v", err)
		return nil
	}

	return records
}

func parseAudioDeviceRecords(data []byte) ([]audioDeviceRecord, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	entries, ok := root["SPAudioDataType"].([]interface{})
	if !ok {
		return nil, nil
	}

	records := make([]audioDeviceRecord, 0)
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		items, ok := entryMap["_items"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			name := getStringValue(itemMap, "_name", "name")
			if name == "" {
				continue
			}

			transport := getStringValue(itemMap,
				"coreaudio_device_transport",
				"coreaudio_transport",
				"transport",
				"coreaudio_device_interface",
			)

			_, isDefault := mapHasTruthyValue(itemMap,
				"coreaudio_device_is_default_output",
				"coreaudio_default_audio_output_device",
				"coreaudio_default_output_device",
				"default_output_device",
				"coreaudio_is_default_output",
			)

			foundConnected, isConnected := mapHasTruthyValue(itemMap,
				"coreaudio_device_is_alive",
				"device_is_alive",
				"device_active",
				"device_is_connected",
				"connected",
				"device_connected",
			)
			if !foundConnected {
				isConnected = true
			}

			records = append(records, audioDeviceRecord{
				Name:            name,
				Transport:       transport,
				IsDefaultOutput: isDefault,
				IsConnected:     isConnected,
			})
		}
	}

	return records, nil
}

func getStringValue(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case string:
				s := strings.TrimSpace(v)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func mapHasTruthyValue(m map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case bool:
				return true, v
			case string:
				n := strings.ToLower(strings.TrimSpace(v))
				switch n {
				case "yes", "true", "1", "on", "spaudio_yes", "enabled":
					return true, true
				case "no", "false", "0", "off", "spaudio_no", "disabled":
					return true, false
				}
			case float64:
				return true, v != 0
			}
		}
	}
	return false, false
}

func getAllConnectedExternalDevices(records []audioDeviceRecord) map[string]bool {
	devices := make(map[string]bool)
	for _, rec := range records {
		if !rec.IsConnected {
			continue
		}

		deviceType := detectDeviceType(rec.Name, rec.Transport)
		if isExternalDevice(deviceType) {
			devices[rec.Name] = true
			continue
		}

		if deviceType != AudioDeviceBuiltIn && isExternalDeviceName(rec.Name) {
			devices[rec.Name] = true
		}
	}

	return devices
}

func isExternalDeviceName(name string) bool {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "bluetooth") ||
		strings.Contains(nameLower, "airpods") ||
		strings.Contains(nameLower, "beats") ||
		strings.Contains(nameLower, "sony") ||
		strings.Contains(nameLower, "bose") ||
		strings.Contains(nameLower, "jabra") ||
		strings.Contains(nameLower, "sennheiser") ||
		strings.Contains(nameLower, "jbl") ||
		strings.Contains(nameLower, "marshall") ||
		strings.Contains(nameLower, "wireless") ||
		strings.Contains(nameLower, "usb") ||
		strings.Contains(nameLower, "hdmi") ||
		strings.Contains(nameLower, "displayport") ||
		strings.Contains(nameLower, "headphone") ||
		strings.Contains(nameLower, "headset") ||
		strings.Contains(nameLower, "external") {
		// Exclude built-in devices
		if !strings.Contains(nameLower, "built-in") &&
			!strings.Contains(nameLower, "internal") &&
			!strings.Contains(nameLower, "macbook") &&
			!strings.Contains(nameLower, "imac") &&
			!strings.Contains(nameLower, "mac mini") &&
			!strings.Contains(nameLower, "mac pro") {
			return true
		}
	}

	return false
}

func deviceNameMatches(name1, name2 string) bool {
	lower1 := strings.ToLower(name1)
	lower2 := strings.ToLower(name2)

	if lower1 == lower2 {
		return true
	}

	if strings.Contains(lower1, lower2) || strings.Contains(lower2, lower1) {
		return true
	}

	return false
}

func (m *AudioMonitor) GetCurrentDevice() *AudioDeviceInfo {
	if !m.supported {
		return nil
	}

	return m.getCurrentAudioDevice()
}
