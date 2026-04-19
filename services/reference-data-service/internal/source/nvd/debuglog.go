package nvd

import (
	"encoding/json"
	"os"
	"time"
)

const debugLogPath = "/Users/tvkhananiev/Documents/MEPHI/ВКР/vkr_temp_arch/.cursor/debug-cf16da.log"

func debugLogNVD(hypothesisID, location, message string, data map[string]any) {
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(map[string]any{
		"sessionId":    "cf16da",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	})
}
